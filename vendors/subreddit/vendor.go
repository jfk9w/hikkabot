package subreddit

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/3rdparty/viddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/vendors"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Vendor struct {
	Clock         flu.Clock
	Storage       Storage
	CleanInterval time.Duration
	FreshThingTTL time.Duration
	RedditClient  *reddit.Client
	MediaManager  *feed.MediaManager
	VidditClient  *viddit.Client
	work          flu.WaitGroup
	cancel        func()
}

func (v *Vendor) DeleteStaleThingsInBackground(ctx context.Context, every time.Duration) error {
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
		_ = queue.Cancel(ctx, err)
		return
	}

	sort.Sort(thingSorter(things))
	percentile := -1
	dirty := true
	for _, thing := range things {
		writeHTML, err := v.processThing(ctx, queue.Header, data, log, &percentile, thing.Data)
		if err != nil {
			_ = queue.Cancel(ctx, err)
			return
		}

		if writeHTML == nil {
			continue
		}

		if dirty {
			now := v.Clock.Now()
			if now.Sub(time.Unix(data.LastCleanSecs, 0)) >= v.CleanInterval {
				freshIDs, err := v.Storage.GetFreshThingIDs(ctx, data.Subreddit, data.SentIDs)
				if err != nil {
					_ = queue.Cancel(ctx, errors.Wrap(err, "get fresh things"))
					return
				}

				data.SentIDs = freshIDs
				data.LastCleanSecs = now.Unix()
			}

			dirty = false
		}

		data.SentIDs.Add(thing.Data.ID)
		if err := queue.Proceed(ctx, writeHTML, data); err != nil {
			log.Warnf("interrupted refresh: %s", err)
			return
		}
	}
}

func (v *Vendor) processThing(ctx context.Context,
	header *feed.Header, data *Data, log *logrus.Entry,
	percentile *int, thing *reddit.ThingData) (
	writeHTML feed.WriteHTML, err error) {

	thing.LastSeen = v.Clock.Now()
	if err := v.Storage.SaveThing(ctx, thing); err != nil {
		return nil, errors.Wrapf(err, "save thing %d", thing.ID)
	}

	log = log.WithField("thing", thing.ID)

	if data.SentIDs.Has(thing.ID) {
		log.Trace("skip: already sent")
		return nil, nil
	}

	if *percentile < 0 {
		var err error
		*percentile, err = v.Storage.GetPercentile(ctx, data.Subreddit, data.Top)
		if err != nil {
			return nil, errors.Wrapf(err, "get %.2f percentile for %s", data.Top, data.Subreddit)
		}
	}

	if thing.Ups < *percentile {
		log.Tracef("skip: ups %d < percentile %d", thing.Ups, *percentile)
		return nil, nil
	}

	if thing.IsSelf {
		if data.MediaOnly {
			return nil, nil
		}

		return func(html *richtext.HTMLWriter) error {
			writeHTMLPrefix(html, data.IndexUsers, thing).
				Bold(thing.Title).Text("\n").
				MarkupString(thing.SelfTextHTML)
			return nil
		}, nil
	}

	media := v.resolveMediaRef(header, thing, data.MediaOnly)
	return func(html *richtext.HTMLWriter) error {
		writeHTMLPrefix(html, data.IndexUsers, thing).
			Text(thing.Title).Text("\n").
			Media(thing.URL, media, true)
		return nil
	}, nil
}

func (v *Vendor) resolveMediaRef(header *feed.Header, thing *reddit.ThingData, mediaOnly bool) richtext.MediaRef {
	ref := &feed.MediaRef{
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
			return vendors.InvalidMediaRef{
				Error: errors.Errorf("failed to find url for %s", thing.URL),
			}
		}
	}

	switch thing.Domain {
	case "gfycat.com", "www.gfycat.com":
		ref.Blob = true
		ref.MediaResolver = resolver.RedGIFs{Site: "gfycat"}
	case "redgifs.com", "www.redgifs.com":
		ref.Blob = true
		ref.MediaResolver = resolver.RedGIFs{Site: "redgifs"}
	case "imgur.com", "www.imgur.com", "i.imgur.com":
		if strings.Contains(ref.URL, ".gifv") {
			ref.URL = strings.Replace(ref.URL, ".gifv", ".mp4", 1)
			ref.MediaResolver = new(feed.DummyMediaResolver)
		} else {
			ref.MediaResolver = new(resolver.Imgur)
		}
	case "youtube.com", "www.youtube.com", "youtu.be":
		ref.MediaResolver = &resolver.YouTube{MediaRef: ref}
	case "preview.redd.it":
		ref.MediaResolver = &feed.DummyMediaResolver{HttpClient: v.RedditClient.HttpClient}
	case "v.redd.it":
		ref.URL = thing.PermalinkURL()
		ref.MediaResolver = resolver.Viddit{Client: v.VidditClient}
	default:
		ref.MediaResolver = new(feed.DummyMediaResolver)
	}

	return v.MediaManager.Submit(ref)
}

func (v *Vendor) getListing(ctx context.Context, subreddit string, limit int) ([]reddit.Thing, error) {
	return v.RedditClient.GetListing(ctx, subreddit, "hot", limit)
}

func (v *Vendor) deleteStaleThings(ctx context.Context, now time.Time) error {
	until := now.Add(-v.FreshThingTTL)
	rowsAffected, err := v.Storage.DeleteStaleThings(ctx, until)
	if err != nil {
		return err
	}

	logrus.Infof("deleted %d stale things", rowsAffected)
	return nil
}

func writeHTMLPrefix(html *richtext.HTMLWriter, indexUsers bool, thing *reddit.ThingData) *richtext.HTMLWriter {
	html = html.
		Text(getSubredditName(thing.Subreddit)).Text(" ").
		Link("💬", thing.PermalinkURL()).Text("\n")
	if indexUsers && thing.Author != "" {
		html = html.Text(`u/`).Text(vendors.Hashtag(thing.Author)).Text("\n")
	}

	return html
}

func getSubredditName(subreddit string) string {
	return "#" + subreddit
}
