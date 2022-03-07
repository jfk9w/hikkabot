package subreddit

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w-go/flu/gormf"

	"github.com/jfk9w-go/flu"
	httpf "github.com/jfk9w-go/flu/httpf"
	tgmedia "github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"hikkabot/3rdparty/reddit"
	"hikkabot/core/feed"
	"hikkabot/core/media"
	"hikkabot/ext/resolvers"
	"hikkabot/util"
)

const Name = "subreddit"

type Vendor struct {
	Context
	Pacing
	CleanDataEvery time.Duration
	FreshThingTTL  time.Duration
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
					if flu.IsContextRelated(err) {
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

var refRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/[ur]/([0-9A-Za-z_]+)$`)

func (v *Vendor) OnResume(ctx context.Context, rawData gormf.JSONB) error {
	var data Data
	if err := rawData.Unmarshal(&data); err != nil {
		return errors.Wrap(err, "unmarshal raw data")
	}

	if err := v.RedditClient.Subscribe(ctx, reddit.Subscribe, []string{data.Subreddit}); err != nil {
		return errors.Wrap(err, "subscribe to subreddit")
	} else {
		logrus.WithField("subreddit", data.Subreddit).Info("subscribe reddit account to subreddit ok")
	}

	return nil
}

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

	data := &Data{Subreddit: subreddit}
	for _, option := range options {
		switch option {
		case "!m":
			data.Layout.HideMedia = true
			data.Layout.ShowText = true
		case "u":
			data.Layout.ShowAuthor = true
		case "p":
			data.Layout.ShowPaywall = true
			data.Layout.HideMediaLink = true
			data.Layout.HideLink = true
			data.Layout.HideTitle = true
		case "l":
			data.Layout.ShowPreference = true
		}
	}

	return &feed.Draft{
		SubID: data.Subreddit,
		Name:  getSubredditName(data.Subreddit),
		Data:  data,
	}, nil
}

func (v *Vendor) Refresh(ctx context.Context, queue *feed.Queue) {
	data := new(Data)
	data.SentIDs = make(util.StringSet)
	if err := queue.GetData(ctx, data); err != nil {
		return
	}

	log := queue.Log(ctx, data)

	things, err := v.getListing(ctx, data.Subreddit, 100)
	if err != nil {
		if errors.As(err, new(net.Error)) {
			log.Warnf("update: failed (network error)")
		} else if errors.As(err, new(*json.SyntaxError)) {
			log.Warnf("update: failed (json error)")
		} else if serr := new(httpf.StatusCodeError); errors.As(err, serr) &&
			(serr.StatusCode < 400 || serr.StatusCode >= 500) {
			log.Warnf("update: failed (http %d)", serr.StatusCode)
		} else {
			_ = queue.Cancel(ctx, err)
		}

		return
	}

	if err := v.Storage.SaveThings(ctx, things); err != nil {
		_ = queue.Cancel(ctx, err)
		return
	}

	now := v.Clock.Now()
	percentile := -1
	dirty := true
	submitted := 0
	for i := range things {
		thing := &things[i]
		writeHTML, err := v.processThing(ctx, now, queue.Header, data, log, &percentile, &thing.Data)
		if err != nil {
			_ = queue.Cancel(ctx, err)
			return
		}

		if writeHTML == nil {
			continue
		}

		if dirty {
			if now.Sub(time.Unix(data.LastCleanSecs, 0)) >= v.CleanDataEvery {
				freshIDs, err := v.Storage.GetFreshThingIDs(ctx, data.SentIDs)
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

		submitted++
		if submitted >= v.MaxBatch {
			return
		}
	}
}

func (v *Vendor) processThing(ctx context.Context, now time.Time,
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
		members, err := v.TelegramClient.GetChatMemberCount(ctx, header.FeedID)
		if err != nil {
			return nil, errors.Wrap(err, "get chat member count")
		}

		pacing := v.Pacing
		sentIDs := data.SentIDs.Slice()
		getPercentile := func(storage Storage) error {
			boost := 0.
			if (data.Layout.ShowPreference || data.Layout.ShowPaywall) && len(sentIDs) > 0 {
				score, err := storage.Score(ctx, header.FeedID, sentIDs)
				if err != nil {
					return errors.Wrap(err, "score")
				}

				log.Debugf("score = %v", score)
				if score.First != nil && now.Sub(*score.First) >= pacing.Stable {
					thingRatio := (float64(score.LikedThings) - float64(score.DislikedThings)) / float64(len(data.SentIDs))
					if members < pacing.MinMembers {
						members = pacing.MinMembers
					}

					userRatio := (float64(score.Likes) - float64(score.Dislikes)) / float64(members)
					boost = pacing.Multiplier * thingRatio * userRatio
					log.Debugf("ur = %f, b = %f", userRatio, boost)
				}
			}

			top := pacing.Base * (boost + 1)
			if top < pacing.Min {
				top = pacing.Min
			}

			v.Metrics.Gauge("top", header.Labels()).Set(top)
			*percentile, err = storage.GetPercentile(ctx, data.Subreddit, top)
			if err != nil {
				return errors.Wrap(err, "get percentile")
			}

			return nil
		}

		if storage, ok := v.Storage.(*SQLStorage); ok {
			err = storage.Unmask().WithContext(ctx).
				Transaction(func(tx *gorm.DB) error { return getPercentile((*SQLStorage)(tx)) })
		} else {
			err = getPercentile(v.Storage)
		}

		if err != nil {
			return nil, errors.Wrap(err, "get percentile")
		}
	}

	log = log.WithField("pct", *percentile)

	if thing.Ups < *percentile {
		log.Debug("update: skip (ups lower than threshold)")
		return nil, nil
	}

	if thing.IsSelf && !data.Layout.ShowText {
		log.Debug("update: skip (hide text)")
		return nil, nil
	}

	if !thing.IsSelf && data.Layout.HideMedia {
		log.Debug("update: skip (hide media)")
		return nil, nil
	}

	writeHTML = v.writeHTML(header, data.Layout, thing)
	return
}

func (v *Vendor) writeHTML(header *feed.Header, layout Layout, thing *reddit.ThingData) feed.WriteHTML {
	var mediaRef tgmedia.Ref
	if !thing.IsSelf && !layout.HideMedia {
		mediaRef = v.createMediaRef(header, thing, !layout.ShowText)
	}

	return layout.WriteHTML(thing, mediaRef)
}

func (v *Vendor) createMediaRef(header *feed.Header, thing *reddit.ThingData, mediaOnly bool) tgmedia.Ref {
	ref := &media.Ref{
		FeedID: header.FeedID,
		URL:    thing.URL.String,
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
		ref.Resolver = media.PlainResolver{v.RedditClient.Client}
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

func getSubredditName(subreddit string) string {
	return "#" + subreddit
}
