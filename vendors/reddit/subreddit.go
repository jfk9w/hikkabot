package reddit

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/pkg/errors"
)

var DefaultThingTTL = 7 * 24 * time.Hour

type Store interface {
	Init(ctx context.Context) (Store, error)
	Thing(ctx context.Context, thing *ThingData) error
	Percentile(ctx context.Context, subreddit string, top float64) (int, error)
	Clean(ctx context.Context, data *SubredditFeedData) (int, error)
}

type SubredditFeedData struct {
	Subreddit     string        `json:"subreddit"`
	SentIDs       flu.Uint64Set `json:"sent_ids,omitempty"`
	Top           float64       `json:"top"`
	LastCleanSecs int64         `json:"last_clean,omitempty"`
	MediaOnly     bool          `json:"media_only,omitempty"`
}

func (d SubredditFeedData) Copy() SubredditFeedData {
	d.SentIDs = d.SentIDs.Copy()
	return d
}

var (
	SubredditFeedRefRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)$`)
	SubredditTable         = goqu.T("reddit")
)

type SubredditFeed struct {
	*Client
	Store        Store
	MediaManager *feed.MediaManager
	Metrics      metrics.Registry
	Viddit       *common.Viddit
}

func (f *SubredditFeed) getListing(ctx context.Context, subreddit string, limit int) ([]Thing, error) {
	return f.Client.GetListing(ctx, subreddit, "hot", limit)
}

func (f *SubredditFeed) getSubredditName(subreddit string) string {
	return "#" + subreddit
}

func (f *SubredditFeed) Parse(ctx context.Context, ref string, options []string) (feed.Candidate, error) {
	groups := SubredditFeedRefRegexp.FindStringSubmatch(ref)
	if len(groups) != 5 {
		return feed.Candidate{}, feed.ErrWrongVendor
	}

	subreddit := groups[4]
	things, err := f.getListing(ctx, subreddit, 1)
	if err != nil {
		return feed.Candidate{}, errors.Wrap(err, "get listing")
	}

	if len(things) > 0 {
		subreddit = things[0].Data.Subreddit
	}

	data := SubredditFeedData{
		Subreddit: subreddit,
		Top:       0.2,
	}

	for _, option := range options {
		switch option {
		case "m":
			data.MediaOnly = true
		default:
			var err error
			data.Top, err = strconv.ParseFloat(option, 64)
			if err != nil || data.Top <= 0 {
				return feed.Candidate{}, errors.Wrap(err, "top must be positive")
			}
		}
	}

	data.SentIDs = make(flu.Uint64Set, int(100*data.Top))
	return feed.Candidate{
		ID:   data.Subreddit,
		Name: f.getSubredditName(data.Subreddit),
		Data: data,
	}, nil
}

func (f *SubredditFeed) newMediaRef(subID feed.SubID, thing ThingData, mediaOnly bool) format.MediaRef {
	ref := &feed.MediaRef{
		FeedID: subID.FeedID,
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
			return common.InvalidMediaRef{
				Error: errors.Errorf("failed to find url for %s", thing.URL),
			}
		}
	}

	f.Metrics.Counter("media", subID.MetricsLabels().Append(
		"domain", thing.Domain,
	)).Inc()

	switch thing.Domain {
	case "gfycat.com", "www.gfycat.com":
		ref.MediaResolver = new(resolver.Gfycat)
	case "redgifs.com", "www.redgifs.com":
		ref.Blob = true
		ref.MediaResolver = new(resolver.RedGIFs)
	case "imgur.com", "www.imgur.com":
		ref.MediaResolver = new(resolver.Imgur)
	case "i.imgur.com":
		if !strings.Contains(ref.URL, ".gifv") {
			ref.MediaResolver = new(resolver.Imgur)
		} else {
			ref.URL = strings.Replace(ref.URL, ".gifv", ".mp4", 1)
			ref.MediaResolver = new(feed.DummyMediaResolver)
		}
	case "youtube.com", "www.youtube.com", "youtu.be":
		ref.MediaResolver = &resolver.YouTube{MediaRef: ref}
	case "preview.redd.it":
		ref.MediaResolver = &feed.DummyMediaResolver{Client: f.Client.Client}
	case "v.redd.it":
		ref.URL = thing.PermalinkURL()
		ref.MediaResolver = resolver.Viddit{Client: f.Viddit}
	default:
		ref.MediaResolver = new(feed.DummyMediaResolver)
	}

	return f.MediaManager.Submit(ref)
}

func (f *SubredditFeed) doLoad(ctx context.Context, rawData feed.Data, queue feed.Queue) error {
	data := &SubredditFeedData{SentIDs: make(flu.Uint64Set)}
	if err := rawData.ReadTo(data); err != nil {
		return errors.Wrap(err, "parse data")
	}

	things, err := f.getListing(ctx, data.Subreddit, 100)
	if err != nil {
		return errors.Wrap(err, "get listing")
	}

	sort.Sort(redditThings(things))
	percentile := -1
	for _, thing := range things {
		thing := thing.Data
		if err := f.Store.Thing(ctx, &thing); err != nil {
			return errors.Wrap(err, "save post")
		}

		if data.SentIDs.Has(thing.ID) {
			continue
		}

		if percentile < 0 {
			percentile, err = f.Store.Percentile(ctx, data.Subreddit, data.Top)
			if err != nil {
				return errors.Wrap(err, "percentile")
			}

			f.Metrics.Gauge("ups", append(queue.SubID.MetricsLabels(),
				"subreddit", data.Subreddit,
				"top", fmt.Sprintf("%.2f", data.Top),
			)).Set(float64(percentile))
		}

		if thing.Ups < percentile {
			continue
		}

		var write feed.WriteHTML
		if thing.IsSelf {
			if data.MediaOnly {
				continue
			} else {
				write = func(html *format.HTMLWriter) error {
					html.Text(f.getSubredditName(data.Subreddit)).Text("\n").
						Bold(thing.Title).Text("\n").
						MarkupString(thing.SelfTextHTML)
					return nil
				}
			}
		} else {
			media := f.newMediaRef(queue.SubID, thing, data.MediaOnly)
			write = func(html *format.HTMLWriter) error {
				html.Text(f.getSubredditName(data.Subreddit)).Text("\n").
					Text(thing.Title).
					Media(thing.URL, media, true)
				return nil
			}
		}

		data.SentIDs.Add(thing.ID)
		f.Store.Clean(ctx, data)
		if !data.SentIDs.Has(thing.ID) {
			continue
		}

		if err := queue.Submit(ctx, feed.Update{
			Write: write,
			Data:  data.Copy(),
		}); err != nil {
			return nil
		}
	}

	return nil
}

func (f *SubredditFeed) Load(ctx context.Context, rawData feed.Data, queue feed.Queue) {
	defer queue.Close()
	if err := f.doLoad(ctx, rawData, queue); err != nil {
		_ = queue.Submit(ctx, feed.Update{Error: err})
	}
}

type redditThings []Thing

func (r redditThings) Len() int {
	return len(r)
}

func (r redditThings) Less(i, j int) bool {
	return r[i].Data.ID < r[j].Data.ID
}

func (r redditThings) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
