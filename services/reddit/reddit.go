package reddit

import (
	"regexp"
	"sort"
	"strconv"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/subscription"
	"github.com/pkg/errors"
)

const thingLimit = 100

func Service() subscription.Item {
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

func (s *Subscription) Parse(ctx subscription.ApplicationContext, cmd string, opts string) error {
	groups := re.FindStringSubmatch(cmd)
	if len(groups) != 7 {
		return subscription.ErrParseFailed
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

func (s *Subscription) Update(ctx subscription.ApplicationContext, offset int64, session *subscription.UpdateQueue) {
	things, err := ctx.RedditClient.GetListing(s.Subreddit, s.Sort, thingLimit)
	if err != nil {
		session.Fail(err)
		return
	}
	sort.Sort(listing(things))
	for i := range things {
		thing := &things[i]
		if thing.Data.Created.Unix() <= offset || thing.Data.Ups < s.MinUps || thing.Data.URL == "" {
			continue
		}
		var media []media.Download
		if thing.Data.URL != "" {
			media = ctx.MediaManager.Download(Media{thing, ctx.RedditClient})
		}
		update := subscription.Update{
			Offset: thing.Data.Created.Unix(),
			Text: format.NewHTML(telegram.MaxCaptionSize, 1, nil, nil).
				Text("#" + s.Subreddit).NewLine().
				Text(thing.Data.Title).
				Format(),
			Media: media,
		}
		if !session.Offer(update) {
			return
		}
	}
}
