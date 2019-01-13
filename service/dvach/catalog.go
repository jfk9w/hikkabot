package dvach

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jfk9w-go/hikkabot/api/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/hikkabot/service"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type catalogOptions struct {
	BoardID string `json:"board_id"`
	Query   string `json:"query,omitempty"`
}

type CatalogService struct {
	agg   *service.Aggregator
	dvach *dvach.Client
}

func Catalog(agg *service.Aggregator, c *dvach.Client) *CatalogService {
	svc := &CatalogService{agg, c}
	agg.Add(svc)
	return svc
}

func (svc *CatalogService) ID() string {
	return "2ch/catalog"
}

func (svc *CatalogService) search(boardID string, query *regexp.Regexp) ([]*dvach.Post, error) {
	catalog, err := svc.dvach.GetCatalog(boardID)
	if err != nil {
		return nil, err
	}

	result := make([]*dvach.Post, 0)
	for _, post := range catalog.Threads {
		text := strings.ToLower(post.Comment)
		if query == nil || query.MatchString(text) {
			result = append(result, post)
		}
	}

	posts := dvachCatalogPosts(result)
	sort.Sort(posts)
	result = []*dvach.Post(posts)

	return result, nil
}

var catalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (svc *CatalogService) Subscribe(input string, chat *telegram.Chat, options string) error {
	groups := catalogRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return service.ErrInvalidFormat
	}

	boardID := groups[4]
	query, err := regexp.Compile(options)
	if err != nil {
		return err
	}

	board, err := svc.dvach.GetBoard(boardID)
	if err != nil {
		return err
	}

	return svc.agg.Subscribe(chat,
		svc.ID(),
		fmt.Sprintf("%s/%s", boardID, query),
		board.Name,
		&catalogOptions{
			BoardID: board.ID,
			Query:   query.String(),
		})
}

var maxHtmlChunkSize = telegram.MaxMessageSize * 5 / 7

func (svc *CatalogService) Update(prevOffset int64, optionsFunc service.OptionsFunc, updatePipe *service.UpdatePipe) {
	defer updatePipe.Close()

	options := new(catalogOptions)
	err := optionsFunc(options)
	if err != nil {
		updatePipe.Error(err)
		return
	}

	query := regexp.MustCompile(options.Query)
	posts, err := svc.search(options.BoardID, query)
	if err != nil {
		updatePipe.Error(err)
		return
	}

	for _, post := range posts {
		offset := int64(post.Num)
		if offset <= prevOffset {
			continue
		}

		update := &service.GenericUpdate{
			Text: html.NewBuilder(maxHtmlChunkSize, 1).
				B().Text(post.DateString).EndB().Br().
				Link(post.URL(), "[LINK]").Br().
				Text("---").Br().
				Parse(post.Comment).
				Build()[0],
		}

		if !updatePipe.Submit(update, offset) {
			return
		}
	}
}

type dvachCatalogPosts []*dvach.Post

func (p dvachCatalogPosts) Len() int {
	return len(p)
}

func (p dvachCatalogPosts) Less(i, j int) bool {
	return p[i].Num < p[j].Num
}

func (p dvachCatalogPosts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
