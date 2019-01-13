package reddit

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/jfk9w-go/hikkabot/api/reddit"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/hikkabot/service"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type options struct {
	Subreddit    string      `json:"subreddit"`
	Sort         reddit.Sort `json:"sort"`
	UpsThreshold int         `json:"ups_threshold,omitempty"`
}

type Service struct {
	agg    *service.Aggregator
	fs     service.FileSystemService
	reddit *reddit.Client
}

func Reddit(agg *service.Aggregator, fs service.FileSystemService, reddit *reddit.Client) *Service {
	svc := &Service{agg, fs, reddit}
	agg.Add(svc)
	return svc
}

func (svc *Service) ID() string {
	return "reddit"
}

// hot|new|top must match reddit.*Sort
var redditRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([A-Za-z_]+)(/(hot|new|top))?$`)

func parseRedditInput(input string) (string, string, error) {
	groups := redditRegexp.FindStringSubmatch(input)
	if len(groups) != 7 {
		return "", "", service.ErrInvalidFormat
	}

	return groups[4], groups[6], nil
}

func (svc *Service) Subscribe(input string, chat *telegram.Chat, args string) error {
	subreddit, sort, err := parseRedditInput(input)
	if err != nil {
		return err
	}

	if sort == "" {
		sort = reddit.HotSort
	}

	things, err := svc.reddit.GetListing(subreddit, sort, 1)
	if err != nil {
		return err
	}

	if len(things) == 0 {
		return errors.New("no entries in subreddit")
	}

	upsThreshold := 0
	if args != "" {
		upsThreshold, err = strconv.Atoi(args)
		if err != nil {
			return err
		}
	}

	subreddit = things[0].Data.Subreddit
	return svc.agg.Subscribe(chat, svc.ID(), subreddit, subreddit, &options{
		Subreddit:    subreddit,
		Sort:         sort,
		UpsThreshold: upsThreshold,
	})
}

func (svc *Service) Update(prevOffset int64, optionsFunc service.OptionsFunc, updatePipe *service.UpdatePipe) {
	defer updatePipe.Close()

	options := new(options)
	err := optionsFunc(options)
	if err != nil {
		updatePipe.Error(err)
		return
	}

	things, err := svc.reddit.GetListing(options.Subreddit, options.Sort, 100)
	if err != nil {
		updatePipe.Error(err)
		return
	}

	for _, thing := range things {
		offset := int64(thing.Data.RawCreatedUTC)
		if offset <= prevOffset {
			continue
		}

		if thing.Data.Ups < options.UpsThreshold {
			continue
		}

		update := &service.GenericUpdate{
			Text: html.NewBuilder(telegram.MaxCaptionSize, 1).
				Text("#"+options.Subreddit).Br().
				Link(thing.Data.URL, "[LINK]").Br().
				Text(thing.Data.Title).
				Build()[0],
		}

		resource := svc.fs.NewTempResource()
		if err := svc.reddit.Download(thing, resource); err == nil {
			update.Entity = resource
			switch thing.Data.Extension {
			case "gifv", "gif", "mp4":
				update.Type = service.VideoUpdate
			default:
				update.Type = service.PhotoUpdate
			}
		}

		if !updatePipe.Submit(update, offset) {
			return
		}
	}
}
