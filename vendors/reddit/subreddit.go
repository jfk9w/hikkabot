package reddit

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/doug-martin/goqu/v9"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/pkg/errors"
)

type Store interface {
	Init(ctx context.Context) (Store, error)
	Thing(ctx context.Context, thing *ThingData) error
	Percentile(ctx context.Context, subreddit string, top float64) (int, error)
	Clean(ctx context.Context, data *SubredditFeedData) (int, error)
}

type SubredditFeedData struct {
	Subreddit     string    `json:"subreddit"`
	SentIDs       Uint64Set `json:"sent_ids,omitempty"`
	Top           float64   `json:"top"`
	LastCleanSecs int64     `json:"last_clean,omitempty"`
	MediaOnly     bool      `json:"media_only,omitempty"`
	IndexUsers    bool      `json:"index_users,omitempty"`
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

func (f *SubredditFeed) ParseSub(ctx context.Context, ref string, options []string) (feed.SubDraft, error) {
	groups := SubredditFeedRefRegexp.FindStringSubmatch(ref)
	if len(groups) != 5 {
		return feed.SubDraft{}, feed.ErrWrongVendor
	}

	subreddit := groups[4]
	things, err := f.getListing(ctx, subreddit, 1)
	if err != nil {
		return feed.SubDraft{}, errors.Wrap(err, "get listing")
	}

	if len(things) > 0 {
		subreddit = things[0].Data.Subreddit
	}

	data := SubredditFeedData{
		Subreddit: subreddit,
		Top:       0.2,
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
				return feed.SubDraft{}, errors.Wrap(err, "top must be positive")
			}
		}
	}

	data.SentIDs = make(Uint64Set, int(100*data.Top))
	return feed.SubDraft{
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
	data := &SubredditFeedData{SentIDs: make(Uint64Set)}
	if err := rawData.ReadTo(data); err != nil {
		return errors.Wrap(err, "parse data")
	}

	things, err := f.getListing(ctx, data.Subreddit, 100)

	if err != nil {
		if IsTemporaryError(err) {
			return nil
		} else {
			return errors.Wrap(err, "get listing")
		}
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
					f.writeHTMLPrefix(html, data.IndexUsers, thing).
						Bold(thing.Title).Text("\n").
						MarkupString(thing.SelfTextHTML)
					return nil
				}
			}
		} else {
			media := f.newMediaRef(queue.SubID, thing, data.MediaOnly)
			write = func(html *format.HTMLWriter) error {
				f.writeHTMLPrefix(html, data.IndexUsers, thing).
					Text(thing.Title).Text("\n").
					Media(thing.URL, media, true)
				return nil
			}
		}

		data.SentIDs.Add(thing.ID)
		if removed, err := f.Store.Clean(ctx, data); err != nil {
			log.Printf("[sub > %s] failed to clean posts: %s", queue.SubID, err)
		} else if removed > 0 {
			log.Printf("[sub > %s] cleaned %d old posts", queue.SubID, removed)
		}

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

var TemporaryErrorStatusCodes = map[int]bool{
	http.StatusBadGateway:     true,
	http.StatusGatewayTimeout: true,
}

func IsTemporaryError(err error) bool {
	for {
		if err = errors.Unwrap(err); err != nil {
			if err, ok := err.(fluhttp.StatusCodeError); ok && TemporaryErrorStatusCodes[err.Code] {
				return true
			}

			if err, ok := err.(net.Error); ok && err.Temporary() {
				return true
			}
		}

		break
	}

	return false
}

func (f *SubredditFeed) writeHTMLPrefix(html *format.HTMLWriter, indexUsers bool, thing ThingData) *format.HTMLWriter {
	html = html.Text(f.getSubredditName(thing.Subreddit)).Text("\n")
	if indexUsers && thing.Author != "" {
		html = html.Text(`u/#`).Text(thing.Author).Text("\n")
	}

	return html
}

func (f *SubredditFeed) LoadSub(ctx context.Context, rawData feed.Data, queue feed.Queue) {
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
