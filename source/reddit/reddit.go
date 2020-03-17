package reddit

import (
	"html"
	"log"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	_media "github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/media/descriptor"
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

type Storage interface {
	RedditUpPivot(id feed.ID, percentile float64, period time.Duration) int
	RedditPost(id feed.ID, name string, ups int, sent bool, period time.Duration) bool
}

type Source struct {
	*reddit.Client
	*_media.Tor
	Storage
	Metrics metrics.Client
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
		log.Printf("Failed to pull subreddit listing for %s: %s", pull.ID, err)
		return nil
	}

	sort.Sort(listing(things))
	minUps := item.MinUps
	if minUps > 0 && minUps < 1 {
		minUps = float64(s.RedditUpPivot(pull.ID, minUps, TTL))
		s.Metrics.Gauge("ups_threshold", metrics.Labels{
			"chat":       pull.ID.ChatID.String(),
			"sub":        pull.ID.ID,
			"percentile": strconv.FormatFloat(item.MinUps, 'f', 2, 64),
		}).Set(minUps)
	}

	for _, thing := range things {
		sent := thing.Data.Ups > int(minUps)
		if !s.RedditPost(pull.ID, thing.Data.Name, thing.Data.Ups, sent, TTL) {
			continue
		}

		s.Metrics.Counter("posts", metrics.Labels{
			"chat": pull.ID.ChatID.String(),
			"sub":  pull.ID.ID,
		}).Inc()

		media := make([]*_media.Promise, 0)
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

var mediaOptions = _media.Options{
	Hashable: true,
	//OCR: &_media.OCR{
	//	Languages: []string{"eng"},
	//	Regex:     regexp.MustCompile(`(?is).*?(cake.*?day|sort.*?by.*?new|upvote|updoot).*`),
	//},
}

func (s Source) mediaDescriptor(thing reddit.Thing) (string, _media.Descriptor, error) {
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
