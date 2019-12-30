package reddit

import (
	"html"
	"regexp"
	"sort"
	"strconv"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
)

const ListingThingLimit = 100

type Item struct {
	Subreddit string
	Sort      reddit.Sort
	MinUps    int
}

type Source struct {
	*reddit.Client
}

var re = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)(/(hot|new|top))?$`)

func (Source) ID() string {
	return "Reddit"
}

func (s Source) Draft(command, options string) (*feed.Draft, error) {
	groups := re.FindStringSubmatch(command)
	if len(groups) != 7 {
		return nil, feed.ErrDraftFailed
	}
	item := Item{}
	item.Subreddit, item.Sort = groups[4], groups[6]
	if item.Sort == "" {
		item.Sort = reddit.Hot
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
	return &feed.Draft{
		ID:   item.Subreddit,
		Name: "#" + item.Subreddit,
		Item: feed.ToBytes(item),
	}, nil
}

func (s Source) Pull(pull *feed.UpdatePull) error {
	item := new(Item)
	pull.FromBytes(item)
	things, err := s.GetListing(item.Subreddit, item.Sort, ListingThingLimit)
	if err != nil {
		return err
	}
	sort.Sort(listing(things))
	for i := range things {
		thing := &things[i]
		if thing.Data.Created.Unix() <= pull.Offset || thing.Data.Ups < item.MinUps {
			continue
		}
		media := make([]*media.Media, 0)
		text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
			Text("#" + item.Subreddit).NewLine()
		if thing.Data.IsSelf {
			text.
				Tag("b").Text(thing.Data.Title).EndTag().
				NewLine().NewLine().
				Parse(html.UnescapeString(thing.Data.SelfTextHTML))
		} else {
			s.ResolveMediaURL(thing)
			media = append(media, s.downloadMedia(pull.Media, thing))
			text.Text(thing.Data.Title)
		}
		update := feed.Update{
			Offset: thing.Data.Created.Unix(),
			Text:   text.Format(),
			Media:  media,
		}
		if !pull.Submit(update) {
			break
		}
	}
	return nil
}

func (s Source) downloadMedia(manager *media.Manager, thing *reddit.Thing) *media.Media {
	var in media.SizeAwareReadable
	if thing.Data.ResolvedURL != "" {
		in = &media.HTTPRequest{Request: s.NewRequest().Resource(thing.Data.ResolvedURL).GET()}
	}
	return manager.Submit(thing.Data.URL, thing.Data.Extension, in)
}
