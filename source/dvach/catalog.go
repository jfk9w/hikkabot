package dvach

import (
	"regexp"
	"sort"
	"strings"

	"github.com/jfk9w/hikkabot/source/common"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

type CatalogItem struct {
	Board  string
	Query  common.Query
	Offset int
}

type CatalogSource struct {
	*dvach.Client
	*mediator.Mediator
}

var catalogre = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (CatalogSource) ID() string {
	return "dc"
}

func (CatalogSource) Name() string {
	return "Dvach/Catalog"
}

func (s CatalogSource) Draft(command, options string, rawData feed.RawData) (*feed.Draft, error) {
	groups := catalogre.FindStringSubmatch(command)
	if len(groups) < 6 {
		return nil, feed.ErrDraftFailed
	}
	item := CatalogItem{}
	item.Board = groups[4]
	if options == "" {
		options = ".*"
	}
	var err error
	item.Query.Regexp, err = regexp.Compile(options)
	if err != nil {
		return nil, errors.Wrap(err, "compile regexp")
	}
	catalog, err := s.GetCatalog(item.Board)
	if err != nil {
		return nil, errors.Wrap(err, "get catalog")
	}
	rawData.Marshal(item)
	return &feed.Draft{
		ID:   item.Board + "/" + item.Query.String(),
		Name: catalog.BoardName + " /" + item.Query.String() + "/",
	}, nil
}

func (s CatalogSource) Pull(pull *feed.UpdatePull) error {
	item := new(CatalogItem)
	pull.RawData.Unmarshal(item)
	catalog, err := s.GetCatalog(item.Board)
	if err != nil {
		return errors.Wrap(err, "get catalog")
	}
	results := make([]dvach.Post, 0)
	for _, thread := range catalog.Threads {
		matches := thread.Num > item.Offset
		matches = matches && item.Query.MatchString(strings.ToLower(thread.Comment))
		if matches {
			results = append(results, thread)
		}
	}
	sort.Sort(queryResults(results))
	for _, thread := range results {
		media := make([]*mediator.Future, 0)
		for _, file := range thread.Files {
			media = append(media, s.SubmitMedia(file.URL(),
				&mediatorRequest{s.Client.Client, file}))
			break
		}
		item.Offset = thread.Num
		pull.RawData.Marshal(item)
		update := feed.Update{
			RawData: pull.RawData.Bytes(),
			Text: format.NewHTML(telegram.MaxMessageSize, 0, DefaultSupportedTags, Board(thread.Board)).
				Tag("b").Text(thread.DateString).EndTag().NewLine().
				Link("[link]", thread.URL()).NewLine().
				Text("---").NewLine().
				Parse(thread.Comment).
				Format(),
			Media: media,
		}
		if !pull.Submit(update) {
			break
		}
	}
	return nil
}
