package reddit

import (
	"regexp"
	"sort"
	"strconv"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/html"
	"github.com/jfk9w/hikkabot/service"
	"github.com/pkg/errors"
)

type options struct {
	Subreddit string      `json:"subreddit"`
	Sort      reddit.Sort `json:"sort"`
	MinUps    int         `json:"min_ups,omitempty"`
}

type Service struct {
	agg    *service.Aggregator
	media  *service.MediaService
	reddit *reddit.Client
}

func Reddit(agg *service.Aggregator, media *service.MediaService, reddit *reddit.Client) *Service {
	return &Service{agg, media, reddit}
}

func (svc *Service) ID() service.ID {
	return "reddit"
}

// hot|new|top must match reddit.*Sort
var redditRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([0-9A-Za-z_]+)(/(hot|new|top))?$`)

func parseRedditInput(input string) (string, string, error) {
	groups := redditRegexp.FindStringSubmatch(input)
	if len(groups) != 7 {
		return "", "", service.ErrInvalidFormat
	}

	return groups[4], groups[6], nil
}

func (svc *Service) Subscribe(input string, chat *service.EnrichedChat, args string) error {
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
		Subreddit: subreddit,
		Sort:      sort,
		MinUps:    upsThreshold,
	})
}

func (svc *Service) Update(prevOffset int64, optionsFunc service.OptionsFunc, updatePipe *service.UpdatePipe) {
	defer updatePipe.Close()

	options := new(options)
	err := optionsFunc(options)
	if err != nil {
		updatePipe.Err = err
		return
	}

	things, err := svc.reddit.GetListing(options.Subreddit, options.Sort, 100)
	if err != nil {
		updatePipe.Err = err
		return
	}

	sort.Sort(thingSort(things))

	for _, thing := range things {
		offset := int64(thing.Data.RawCreatedUTC)
		if offset <= prevOffset {
			continue
		}

		if thing.Data.Ups < options.MinUps {
			continue
		}

		var mediaOut chan service.MediaResponse
		if thing.Data.URL != "" {
			mediaOut = make(chan service.MediaResponse)
			go svc.media.Download(mediaOut, service.MediaRequest{
				Func:    svc.mediaFunc(thing),
				Href:    thing.Data.URL,
				MinSize: service.MinMediaSize,
			})
		}

		text := html.NewBuilder(telegram.MaxCaptionSize, 1).
			Text("#" + options.Subreddit).Br().
			Text(thing.Data.Title).
			Build()

		mediaSize := 1
		if thing.Data.URL == "" {
			mediaSize = 0
		}

		update := service.Update{
			Offset:    offset,
			Text:      service.UpdateTextSlice(text),
			MediaSize: mediaSize,
			Media:     mediaOut,
		}

		if !updatePipe.Submit(update) {
			return
		}
	}
}

func (svc *Service) mediaFunc(thing *reddit.Thing) service.MediaFunc {
	return func(resource flu.FileSystemResource) (service.MediaType, error) {
		err := svc.reddit.Download(thing, resource)
		return mediaType(thing), err
	}
}

func mediaType(thing *reddit.Thing) service.MediaType {
	switch thing.Data.Extension {
	case "gifv", "gif", "mp4":
		return service.Video
	default:
		return service.Photo
	}
}

type thingSort []*reddit.Thing

func (t thingSort) Len() int {
	return len(t)
}

func (t thingSort) Less(i, j int) bool {
	return t[i].Data.RawCreatedUTC < t[j].Data.RawCreatedUTC
}

func (t thingSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
