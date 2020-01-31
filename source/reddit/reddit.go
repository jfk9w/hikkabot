package reddit

import (
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w/hikkabot/mediator/request"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

const ListingThingLimit = 100

type Item struct {
	Subreddit string
	Sort      string
	MinUps    int
	Seen      map[string]int64
}

type Source struct {
	*reddit.Client
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
	item.Subreddit, item.Sort = groups[4], groups[6]
	if item.Sort == "" {
		item.Sort = "hot"
	}
	if options != "" {
		var err error
		item.MinUps, err = strconv.Atoi(options)
		if err != nil {
			return nil, errors.Wrap(err, "parse minups")
		}
	}
	things, err := s.GetListing(item.Subreddit, item.Sort, 1)
	if err != nil {
		return nil, errors.Wrap(err, "get listing")
	}
	if len(things) < 1 {
		return nil, errors.New("no entries in /r/" + item.Subreddit)
	}
	rawData.Marshal(item)
	return &feed.Draft{
		ID:   item.Subreddit,
		Name: "#" + item.Subreddit,
	}, nil
}

func (s Source) Pull(pull *feed.UpdatePull) error {
	item := new(Item)
	pull.RawData.Unmarshal(item)
	things, err := s.GetListing(item.Subreddit, item.Sort, ListingThingLimit)
	if err != nil {
		return err
	}
	sort.Sort(listing(things))
	clean := false
	for i := range things {
		thing := &things[i]
		if thing.Data.Ups <= item.MinUps {
			continue
		}
		if _, ok := item.Seen[thing.Data.Name]; ok {
			continue
		}
		media := make([]*mediator.Future, 0)
		text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
			Text("#" + item.Subreddit).NewLine()
		if thing.Data.IsSelf {
			text.
				Tag("b").Text(thing.Data.Title).EndTag().
				NewLine().NewLine().
				Parse(html.UnescapeString(thing.Data.SelfTextHTML))
		} else {
			req, err := s.mediatorRequest(thing)
			if err != nil {
				req = &mediator.FailedRequest{Error: err}
			}
			media = append(media, pull.Mediator.Submit(thing.Data.URL, req))
			text.Text(thing.Data.Title)
		}
		item.Seen[thing.Data.Name] = thing.Data.Created.Unix()
		if !clean {
			now := time.Now()
			for name, created := range item.Seen {
				if now.Sub(time.Unix(created, 0)).Hours() > 24 {
					delete(item.Seen, name)
				}
			}
			clean = true
		}
		pull.RawData.Marshal(item)
		update := feed.Update{
			RawData: pull.RawData.Bytes(),
			Text:    text.Format(),
			Media:   media,
		}
		if !pull.Submit(update) {
			break
		}
	}
	return nil
}

var (
	imagere = regexp.MustCompile(`^.*\.(.*)$`)
	ocrre   = regexp.MustCompile(`(?is).*?(cake.*?day|sort.*?by.*?new).*`)
	ocr     = mediator.OCR{
		Filtered:  true,
		Languages: []string{"eng"},
		Regexp:    ocrre,
	}
)

func (s Source) mediatorRequest(thing *reddit.Thing) (mediator.Request, error) {
	url := thing.Data.URL
	switch thing.Data.Domain {
	case "i.redd.it":
		groups := imagere.FindStringSubmatch(url)
		if len(groups) != 2 {
			return nil, errors.New("unable to find URL")
		} else {
			return &mediator.HTTPRequest{
				URL:    url,
				Format: groups[1],
				OCR:    ocr,
			}, nil
		}
	case "v.redd.it":
		url := getFallbackURL(thing.Data.MediaContainer)
		if url == "" {
			for _, mc := range thing.Data.CrosspostParentList {
				url = getFallbackURL(mc)
				if url != "" {
					break
				}
			}
		}
		if url == "" {
			return nil, errors.New("no fallback URL")
		} else {
			return &mediator.HTTPRequest{
				URL:    url,
				Format: "mp4",
			}, nil
		}
	case "youtube.com", "youtu.be":
		return &request.Youtube{
			URL:     url,
			MaxSize: mediator.MaxSize(telegram.Video)[1],
		}, nil
	case "imgur.com":
		return &request.Imgur{URL: url, OCR: ocr}, nil
	case "gfycat.com":
		return &request.Gfycat{URL: url}, nil
	case "i.imgur.com", "vidble.com":
		url := thing.Data.URL
		dot := strings.LastIndex(url, ".")
		if dot < 0 {
			return nil, errors.Errorf("unable to recognize format of %s", url)
		} else {
			return &mediator.HTTPRequest{
				URL:    url,
				Format: url[dot+1:],
				OCR:    ocr,
			}, nil
		}
	}
	return nil, errors.Errorf("unknown domain: %s", thing.Data.Domain)
}

func getFallbackURL(mc reddit.MediaContainer) string {
	url := mc.Media.RedditVideo.FallbackURL
	if url == "" {
		url = mc.SecureMedia.RedditVideo.FallbackURL
	}
	return url
}
