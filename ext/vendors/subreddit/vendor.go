package subreddit

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	tgmedia "github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/3rdparty/viddit"
	"github.com/jfk9w/hikkabot/core/event"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/jfk9w/hikkabot/core/media"
	"github.com/jfk9w/hikkabot/ext/resolvers"
	"github.com/jfk9w/hikkabot/ext/vendors"
	"github.com/jfk9w/hikkabot/util"
)

var clickCommandKey = "sr_c"

type Vendor struct {
	flu.Clock
	Storage        Storage
	EventStorage   event.Storage
	CleanDataEvery time.Duration
	FreshThingTTL  time.Duration
	RedditClient   *reddit.Client
	MediaManager   *media.Manager
	VidditClient   *viddit.Client
	work           flu.WaitGroup
	cancel         func()
}

func (v *Vendor) ScheduleMaintenance(ctx context.Context, every time.Duration) error {
	if v.cancel != nil {
		return nil
	}

	if err := v.deleteStaleThings(ctx, v.Clock.Now()); err != nil {
		return err
	}

	v.cancel = v.work.Go(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := v.deleteStaleThings(ctx, now); err != nil {
					if ctx.Err() != nil {
						return
					}

					logrus.Warnf("delete stale things: %s", err)
				}
			}
		}
	})

	return nil
}

func (v *Vendor) Close() error {
	if v.cancel != nil {
		v.cancel()
		v.work.Wait()
	}

	return nil
}

var refRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)$`)

func (v *Vendor) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := refRegexp.FindStringSubmatch(ref)
	if len(groups) != 5 {
		return nil, feed.ErrWrongVendor
	}

	subreddit := groups[4]
	things, err := v.getListing(ctx, subreddit, 1)
	if err != nil {
		return nil, errors.Wrap(err, "get listing")
	}

	if len(things) > 0 {
		subreddit = things[0].Data.Subreddit
	}

	data := &Data{
		Subreddit: subreddit,
		Top:       0.3,
		MediaOnly: true,
	}

	for _, option := range options {
		switch option {
		case "!m":
			data.MediaOnly = false
		case "u":
			data.IndexUsers = true
		case "t":
			data.TrackClicks = true
		default:
			var err error
			data.Top, err = strconv.ParseFloat(option, 64)
			if err != nil || data.Top <= 0 {
				return nil, errors.Wrap(err, "top must be positive")
			}
		}
	}

	data.SentIDs = make(util.Uint64Set)
	return &feed.Draft{
		SubID: data.Subreddit,
		Name:  getSubredditName(data.Subreddit),
		Data:  data,
	}, nil
}

func (v *Vendor) Refresh(ctx context.Context, queue *feed.Queue) {
	data := new(Data)
	data.SentIDs = make(util.Uint64Set)
	if err := queue.GetData(ctx, data); err != nil {
		return
	}

	log := queue.Log(ctx, data)

	things, err := v.getListing(ctx, data.Subreddit, 100)
	if err != nil {
		switch err := err.(type) {
		case net.Error:
			log.Warnf("update: failed (network error)")
		case *json.SyntaxError:
			log.Warnf("update: failed (json error)")
		case fluhttp.StatusCodeError:
			if err.StatusCode >= 400 && err.StatusCode < 500 {
				_ = queue.Cancel(ctx, err)
			} else {
				log.Warnf("update: failed (http %d)", err.StatusCode)
			}

		default:
			_ = queue.Cancel(ctx, err)
		}

		return
	}

	if err := v.Storage.SaveThings(ctx, things); err != nil {
		_ = queue.Cancel(ctx, err)
		return
	}

	percentile := -1
	dirty := true
	for i := range things {
		thing := &things[i]
		writeHTML, err := v.processThing(ctx, queue.Header, data, log, &percentile, &thing.Data)
		if err != nil {
			_ = queue.Cancel(ctx, err)
			return
		}

		if writeHTML == nil {
			continue
		}

		if dirty {
			now := v.Clock.Now()
			if now.Sub(time.Unix(data.LastCleanSecs, 0)) >= v.CleanDataEvery {
				freshIDs, err := v.Storage.GetFreshThingIDs(ctx, data.Subreddit, data.SentIDs)
				if err != nil {
					_ = queue.Cancel(ctx, errors.Wrap(err, "get fresh things"))
					return
				}

				staleIDs := len(data.SentIDs) - len(freshIDs)
				if staleIDs > 0 {
					log.Infof("removed %d stale things from data", staleIDs)
				}

				data.SentIDs = freshIDs
				data.LastCleanSecs = now.Unix()
			}

			dirty = false
		}

		data.SentIDs.Add(thing.Data.ID)
		if err := queue.Proceed(ctx, writeHTML, data); err != nil {
			return
		}
	}
}

func (v *Vendor) processThing(ctx context.Context,
	header *feed.Header, data *Data, log *logrus.Entry,
	percentile *int, thing *reddit.ThingData) (
	writeHTML feed.WriteHTML, err error) {

	log = log.WithFields(logrus.Fields{
		"thing": thing.ID,
		"ups":   thing.Ups,
	})

	if data.SentIDs.Has(thing.ID) {
		log.Debugf("update: skip (already sent)")
		return nil, nil
	}

	if *percentile < 0 {
		var err error
		*percentile, err = v.Storage.GetPercentile(ctx, data.Subreddit, data.Top)
		if err != nil {
			return nil, errors.Wrapf(err, "get %.2f percentile for %s", data.Top, data.Subreddit)
		}
	}

	log = log.WithField("pct", *percentile)

	if thing.Ups < *percentile {
		log.Debug("update: skip (ups lower than threshold)")
		return nil, nil
	}

	if thing.IsSelf && data.MediaOnly {
		log.Debug("update: skip (media only)")
		return nil, nil
	}

	return v.writeHTML(header, data, thing)
}

func (v *Vendor) writeHTML(header *feed.Header, data *Data, thing *reddit.ThingData) (feed.WriteHTML, error) {
	if thing.IsSelf {
		return func(html *html.Writer) error {
			writeHTMLPrefix(html, data.IndexUsers, false, thing).
				Bold(thing.Title).Text("\n").
				MarkupString(thing.SelfTextHTML)
			return nil
		}, nil
	}

	mediaRef := v.createMediaRef(header, thing, data.MediaOnly)
	return func(html *html.Writer) error {
		writeHTMLPrefix(html, data.IndexUsers, data.TrackClicks, thing).
			Text(thing.Title).Text("\n").
			Media(thing.URL, mediaRef, true, !data.TrackClicks)
		return nil
	}, nil
}

func (v *Vendor) createMediaRef(header *feed.Header, thing *reddit.ThingData, mediaOnly bool) tgmedia.Ref {
	ref := &media.Ref{
		FeedID: header.FeedID,
		URL:    thing.URL,
		Dedup:  mediaOnly,
	}

	if thing.Domain == "v.redd.it" {
		ref.URL = thing.MediaContainer.FallbackURL()
		if ref.URL == "" {
			for _, mc := range thing.CrosspostParentList {
				ref.URL = mc.FallbackURL()
				if ref.URL != "" {
					break
				}
			}
		}

		if ref.URL == "" {
			ref.Resolver = media.ErrorResolver("url is empty")
			return ref
		}
	}

	switch thing.Domain {
	case "gfycat.com", "www.gfycat.com":
		ref.Blob = true
		ref.Resolver = resolvers.Gfycat("gfycat")
	case "redgifs.com", "www.redgifs.com":
		ref.Blob = true
		ref.Resolver = resolvers.Gfycat("redgifs")
	case "imgur.com", "www.imgur.com", "i.imgur.com":
		if strings.Contains(ref.URL, ".gifv") {
			ref.URL = strings.Replace(ref.URL, ".gifv", ".mp4", 1)
			ref.Resolver = media.PlainResolver{}
		} else {
			ref.Resolver = new(resolvers.Imgur)
		}
	case "preview.redd.it":
		ref.Resolver = media.PlainResolver{HttpClient: v.RedditClient.HttpClient}
	case "v.redd.it":
		ref.URL = thing.PermalinkURL()
		ref.Resolver = (*resolvers.Viddit)(v.VidditClient)
	default:
		ref.Resolver = media.PlainResolver{}
	}

	return v.MediaManager.Submit(ref)
}

func (v *Vendor) getListing(ctx context.Context, subreddit string, limit int) ([]reddit.Thing, error) {
	things, err := v.RedditClient.GetListing(ctx, subreddit, "hot", limit)
	if err != nil {
		return nil, err
	}

	now := v.Clock.Now()
	for i := range things {
		things[i].LastSeen = now
	}

	sort.Sort(thingSorter(things))
	return things, nil
}

func (v *Vendor) deleteStaleThings(ctx context.Context, now time.Time) error {
	until := now.Add(-v.FreshThingTTL)
	rowsAffected, err := v.Storage.DeleteStaleThings(ctx, until)
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		logrus.Infof("deleted %d stale things", rowsAffected)
	}

	return nil
}

func writeHTMLPrefix(html *html.Writer, indexUsers bool, trackClicks bool, thing *reddit.ThingData) *html.Writer {
	html = html.Text(getSubredditName(thing.Subreddit))
	var buttons []telegram.Button
	if out, ok := html.Out.(*output.Paged); ok && trackClicks {
		if chat, ok := out.Receiver.(*receiver.Chat); ok {
			buttons = []telegram.Button{
				(&telegram.Command{
					Key:  clickCommandKey,
					Args: []string{thing.Subreddit, thing.Name},
				}).Button("full post with links in PM"),
			}

			chat.ReplyMarkup = telegram.InlineKeyboard(buttons)
			out.PageCount = 1
			out.PageSize = telegram.MaxCaptionSize
			html.Text("\n")
		}
	}

	if len(buttons) == 0 {
		html = html.Text(" ").Link("💬", thing.PermalinkURL()).Text("\n")
		if indexUsers && thing.Author != "" {
			html = html.Text(`u/`).Text(vendors.Hashtag(thing.Author)).Text("\n")
		}
	}

	return html
}

func getSubredditName(subreddit string) string {
	return "#" + subreddit
}
