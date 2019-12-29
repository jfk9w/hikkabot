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

const thingLimit = 100

func Service() feed.Item {
	return new(Subscription)
}

type Subscription struct {
	Subreddit string
	Sort      reddit.Sort
	MinUps    int
}

func (s *Subscription) Service() string {
	return "Reddit"
}

func (s *Subscription) ID() string {
	return s.Subreddit
}

func (s *Subscription) Name() string {
	return "#" + s.Subreddit
}

var re = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)(/(hot|new|top))?$`)

func (s *Subscription) Parse(ctx feed.Context, cmd string, opts string) error {
	groups := re.FindStringSubmatch(cmd)
	if len(groups) != 7 {
		return feed.ErrParseFailed
	}
	subreddit, sort := groups[4], groups[6]
	if sort == "" {
		sort = reddit.Hot
	}
	minUps := 0
	if opts != "" {
		var err error
		minUps, err = strconv.Atoi(opts)
		if err != nil {
			return errors.Wrap(err, "on minups conversion")
		}
	}
	things, err := ctx.RedditClient.GetListing(subreddit, sort, 1)
	if err != nil {
		return errors.Wrap(err, "on listing")
	}
	if len(things) < 1 {
		return errors.New("no entries in /r/" + subreddit)
	}
	s.Subreddit = subreddit
	s.Sort = sort
	s.MinUps = minUps
	return nil
}

func (s *Subscription) Update(ctx feed.Context, offset int64, session *feed.UpdateQueue) error {
	things, err := ctx.RedditClient.GetListing(s.Subreddit, s.Sort, thingLimit)
	if err != nil {
		return err
	}
	sort.Sort(listing(things))
	for i := range things {
		thing := &things[i]
		if thing.Data.Created.Unix() <= offset || thing.Data.Ups < s.MinUps || thing.Data.URL == "" {
			continue
		}
		media := make([]*media.Media, 0)
		text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
			Text("#" + s.Subreddit).NewLine()
		if thing.Data.IsSelf {
			text.
				Tag("b").Text(thing.Data.Title).EndTag().
				NewLine().NewLine().
				Parse(html.UnescapeString(thing.Data.SelfTextHTML))
		} else {
			ctx.RedditClient.ResolveMediaURL(thing)
			media = append(media, downloadMedia(ctx, thing))
			text.Text(thing.Data.Title)
		}
		update := feed.Update{
			Offset: thing.Data.Created.Unix(),
			Text:   text.Format(),
			Media:  media,
		}
		if !session.Offer(update) {
			break
		}
	}
	return nil
}

func downloadMedia(ctx feed.Context, thing *reddit.Thing) *media.Media {
	var in media.SizeAwareReadable
	if thing.Data.ResolvedURL != "" {
		in = &media.HTTPRequest{Request: ctx.RedditClient.NewRequest().Resource(thing.Data.ResolvedURL).GET()}
	}
	return ctx.MediaManager.Submit(thing.Data.URL, thing.Data.Extension, in)
}
