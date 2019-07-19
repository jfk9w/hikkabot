package reddit

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/app/media"
	"github.com/jfk9w/hikkabot/app/subscription"
	"github.com/jfk9w/hikkabot/html"
	"github.com/pkg/errors"
)

const thingLimit = 100

func Factory() subscription.Interface {
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

var re = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)(/(hot|new|top))?$`)

func (s *Subscription) Parse(ctx subscription.Context, cmd string, opts string) (string, error) {
	groups := re.FindStringSubmatch(cmd)
	if len(groups) != 7 {
		return subscription.EmptyHash, subscription.ErrParseFailed
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
			return subscription.EmptyHash, errors.Wrap(err, "on minups conversion")
		}
	}

	things, err := ctx.RedditClient.GetListing(subreddit, sort, 1)
	if err != nil {
		return subscription.EmptyHash, errors.Wrap(err, "on listing")
	}

	if len(things) < 1 {
		return subscription.EmptyHash, errors.New("no entries in /r/" + subreddit)
	}

	s.Subreddit = subreddit
	s.Sort = sort
	s.MinUps = minUps

	return fmt.Sprintf("%s/%s/%d", subreddit, sort, minUps), nil
}

func (s *Subscription) Update(ctx subscription.Context, offset subscription.Offset, uc *subscription.UpdateCollection) {
	defer close(uc.C)
	things, err := ctx.RedditClient.GetListing(s.Subreddit, s.Sort, thingLimit)
	if err != nil {
		uc.Err = err
		return
	}

	for i := range things {
		thing := &things[i]
		o := subscription.Offset(thing.Data.Created.Unix())
		if o <= offset || thing.Data.Ups < s.MinUps || thing.Data.URL == "" {
			continue
		}

		me := []media.Media{{
			Href:    thing.Data.URL,
			Factory: s.mediaFactory(ctx, thing),
		}}
		ctx.MediaManager.Download(me)

		update := subscription.Update{
			Offset: o,
			Text: html.NewBuilder(telegram.MaxCaptionSize, 1).
				Text("#" + s.Subreddit).Br().
				Text(thing.Data.Title).
				Build(),
			Media: me,
		}

		select {
		case <-uc.Interrupt():
			return
		case uc.C <- update:
			continue
		}
	}
}

func (s *Subscription) mediaFactory(ctx subscription.Context, thing *reddit.Thing) media.Factory {
	return func(resource flu.FileSystemResource) (media.Type, error) {
		err := ctx.RedditClient.Download(thing, resource)
		if err != nil {
			return 0, err
		}

		return mediaType(thing), nil
	}
}

func mediaType(thing *reddit.Thing) media.Type {
	switch thing.Data.Extension {
	case "gifv", "gif", "mp4":
		return media.Video
	default:
		return media.Photo
	}
}
