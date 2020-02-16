package reddit

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jfk9w/hikkabot/media/descriptor"

	"github.com/jfk9w/hikkabot/media"

	"github.com/jfk9w/hikkabot/metrics"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	"github.com/pkg/errors"
)

const (
	ListingThingLimit = 100
	TTL               = 2 * 24 * time.Hour // 2 days
)

type Item struct {
	Sort   string
	MinUps float64
}

type Event struct {
	Name string
	Ups  int
	Seen bool
}

type Storage interface {
	feed.LogStorage
	Events(id feed.ID, period time.Duration) []feed.RawData
}

type Source struct {
	*reddit.Client
	*media.Tor
	Storage
	metrics.Metrics
}

var re = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)(/(hot|new|top))?$`)

func (Source) ID() string {
	return "r"
}

func (Source) Name() string {
	return "Reddit"
}

func (s Source) Draft(command, options string, rawData feed.RawData) (*feed.Draft, error) {
	groups := re.FindStringSubmatch(command)
	if len(groups) != 7 {
		return nil, feed.ErrDraftFailed
	}
	item := Item{}
	item.Sort = groups[6]
	if item.Sort == "" {
		item.Sort = "hot"
	}
	if options != "" {
		var err error
		item.MinUps, err = strconv.ParseFloat(options, 64)
		if err != nil {
			return nil, errors.Wrap(err, "parse MinUps")
		}
	}
	subreddit := groups[4]
	things, err := s.GetListing(subreddit, item.Sort, 1)
	if err != nil {
		return nil, errors.Wrap(err, "get listing")
	}
	if len(things) < 1 {
		return nil, errors.New("no entries in /r/" + subreddit)
	}
	rawData.Marshal(item)
	subreddit = things[0].Data.Subreddit
	return &feed.Draft{
		ID:   subreddit,
		Name: "#" + subreddit,
	}, nil
}

func (s Source) Pull(pull *feed.UpdatePull) error {
	item := new(Item)
	pull.RawData.Unmarshal(item)
	things, err := s.GetListing(pull.ID.ID, item.Sort, ListingThingLimit)
	if err != nil {
		return err
	}
	sort.Sort(listing(things))
	minUps, events := s.collectEvents(pull.ID, item.MinUps)
	for i := range things {
		thing := things[i]
		event := events[thing.Data.Name]
		s.Log(pull.ID, feed.NewRawData(nil).
			Marshal(Event{
				Name: thing.Data.Name,
				Ups:  thing.Data.Ups,
				Seen: event.Seen || thing.Data.Ups >= minUps,
			}),
		)

		if event.Seen {
			continue
		}

		s.Counter("posts", metrics.Labels{
			"chat": pull.ID.ChatID.String(),
			"sub":  pull.ID.ID,
		}).Inc()

		if thing.Data.Ups < minUps {
			continue
		}

		media := make([]*media.Promise, 0)
		text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
			Text(pull.Name).NewLine()
		if thing.Data.IsSelf {
			text.
				Tag("b").Text(thing.Data.Title).EndTag().
				NewLine().NewLine().
				Parse(html.UnescapeString(thing.Data.SelfTextHTML))
		} else {
			url, md, err := s.mediaDescriptor(thing)
			if err == nil {
				media = append(media, s.Submit(url, md, mediaOptions))
			}

			text.Text(thing.Data.Title)
		}

		update := feed.Update{
			RawData: pull.RawData.Bytes(),
			Pages:   text.Format().Pages,
			Media:   media,
		}

		if !pull.Submit(update) {
			break
		}
	}
	return nil
}

func (s Source) collectEvents(id feed.ID, minUps float64) (int, map[string]Event) {
	raw := s.Events(id, TTL)
	events := make(map[string]Event)
	for _, rawData := range raw {
		var event Event
		rawData.Unmarshal(&event)
		prev := events[event.Name]
		if prev.Ups > event.Ups {
			event.Ups = prev.Ups
		}
		event.Seen = event.Seen || prev.Seen
		events[event.Name] = event
	}

	quantile := int(minUps)
	if len(events) > 0 && minUps < 1 {
		ups := make([]int, 0)
		for _, v := range events {
			ups = append(ups, v.Ups)
		}

		sort.Ints(ups)
		quantile = ups[int(float64(len(ups))*minUps)]
		s.Gauge("ups", metrics.Labels{
			"chat":     id.ChatID.String(),
			"sub":      id.ID,
			"quantile": fmt.Sprintf("%.2f", minUps),
		}).Set(float64(quantile))
	}

	return quantile, events
}

var mediaOptions = media.Options{
	Hashable: true,
	OCR: &media.OCR{
		Languages: []string{"eng"},
		Regex:     regexp.MustCompile(`(?is).*?(cake.*?day|sort.*?by.*?new|upvote|updoot).*`),
	},
}

func (s Source) mediaDescriptor(thing reddit.Thing) (string, media.Descriptor, error) {
	url := thing.Data.URL
	if thing.Data.Domain == "v.redd.it" {
		url = getFallbackURL(thing.Data.MediaContainer)
		if url == "" {
			for _, mc := range thing.Data.CrosspostParentList {
				url = getFallbackURL(mc)
				if url != "" {
					break
				}
			}
		}

		if url == "" {
			return "", nil, errors.New("no fallback URL")
		}
	}

	md, err := descriptor.From(s.Client.Client, url)
	return url, md, err
}

func getFallbackURL(mc reddit.MediaContainer) string {
	url := mc.Media.RedditVideo.FallbackURL
	if url == "" {
		url = mc.SecureMedia.RedditVideo.FallbackURL
	}
	return url
}
