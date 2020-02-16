package dvach

import (
	"regexp"
	"sort"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	_media "github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/source/common"
	"github.com/pkg/errors"
)

type CatalogItem struct {
	Board  string
	Query  common.Query
	Offset int
}

type CatalogSource struct {
	*dvach.Client
	*_media.Tor
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
		media := make([]*_media.Promise, 0)
		for _, file := range thread.Files {
			media = append(media, s.Submit(file.URL(),
				&mediaDescriptor{s.Client.Client, file},
				_media.Options{
					Hashable: false,
					Buffer:   true,
				},
			))

			break
		}

		item.Offset = thread.Num
		pull.RawData.Marshal(item)

		update := feed.Update{
			RawData: pull.RawData.Bytes(),
			Pages: format.NewHTML(telegram.MaxMessageSize, 0, DefaultSupportedTags, Board(thread.Board)).
				Tag("b").Text(thread.DateString).EndTag().NewLine().
				Link("[link]", thread.URL()).NewLine().
				Text("---").NewLine().
				Parse(thread.Comment).
				Format().Pages,
			Media: media,
		}

		if !pull.Submit(update) {
			break
		}
	}

	return nil
}
