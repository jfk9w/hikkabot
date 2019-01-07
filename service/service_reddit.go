package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jfk9w-go/hikkabot/api/reddit"
	"github.com/jfk9w-go/hikkabot/common"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type RedditOptions struct {
	Subreddit    string      `json:"subreddit"`
	Sort         reddit.Sort `json:"sort"`
	UpsThreshold int         `json:"ups_threshold,omitempty"`
}

type RedditService struct {
	BaseSubscribeService
	FileSystemService
	c *reddit.Client
}

func Reddit(base BaseSubscribeService, fs FileSystemService, c *reddit.Client) *RedditService {
	base.Type = RedditType
	return &RedditService{
		BaseSubscribeService: base,
		FileSystemService:    fs,
		c:                    c,
	}
}

// hot|new|top must match reddit.*Sort
var redditRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/r/([A-Za-z_]+)(/(hot|new|top))?$`)

func parseRedditInput(input string) (string, string, error) {
	var groups = redditRegexp.FindStringSubmatch(input)
	if len(groups) != 7 {
		return "", "", ErrInvalidFormat
	}

	return groups[4], groups[6], nil
}

func (svc *RedditService) Subscribe(input string, chatID telegram.ID, options string) error {
	subreddit, sort, err := parseRedditInput(input)
	if err != nil {
		return err
	}

	if sort == "" {
		sort = reddit.HotSort
	}

	things, err := svc.c.GetListing(subreddit, sort, 1)
	if err != nil {
		return err
	}

	if len(things) == 0 {
		return errors.New("no entries in subreddit")
	}

	upsThreshold := 0
	if options != "" {
		upsThreshold, err = strconv.Atoi(options)
		if err != nil {
			return err
		}
	}

	subreddit = things[0].Data.Subreddit
	return svc.subscribe(chatID, subreddit, subreddit+"/"+sort, &RedditOptions{
		Subreddit:    subreddit,
		Sort:         sort,
		UpsThreshold: upsThreshold,
	})
}

func (svc *RedditService) Update(currentOffset Offset, rawOptions RawOptions, feed *Feed) {
	defer feed.CloseIn()

	options := new(RedditOptions)
	err := svc.readOptions(rawOptions, options)
	if err != nil {
		feed.Error(err)
		return
	}

	things, err := svc.c.GetListing(options.Subreddit, options.Sort, 100)
	if err != nil {
		feed.Error(err)
		return
	}

	for _, thing := range things {
		offset := Offset(thing.Data.RawCreatedUTC)
		if offset <= currentOffset {
			continue
		}

		if thing.Data.Ups < options.UpsThreshold {
			continue
		}

		text := fmt.Sprintf("%s\n%s", common.Link(thing.Data.URL, "[LINK]"), thing.Data.Title)
		ok := false
		resource := svc.newTempResource()
		err := svc.c.Download(thing, resource)
		if err == nil {
			var fun func(interface{}, string, Offset) bool
			switch thing.Data.Extension {
			case "gifv", "gif", "mp4":
				fun = feed.SubmitVideo

			default:
				fun = feed.SubmitPhoto
			}

			ok = fun(resource, text, offset)
		}

		if err != nil {
			ok = feed.SubmitText(text, false, offset)
		}

		if !ok {
			return
		}
	}
}
