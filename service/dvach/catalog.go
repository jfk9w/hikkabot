package dvach

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/html"
	"github.com/jfk9w/hikkabot/service"
)

type catalogOptions struct {
	BoardID string `json:"board_id"`
	Query   string `json:"query,omitempty"`
}

type CatalogService struct {
	agg   *service.Aggregator
	fs    service.FileSystemService
	dvach *dvach.Client
}

func Catalog(agg *service.Aggregator, fs service.FileSystemService, c *dvach.Client) *CatalogService {
	return &CatalogService{agg, fs, c}
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

func (svc *CatalogService) Subscribe(input string, chat *service.EnrichedChat, options string) error {
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
		fmt.Sprintf("%s /%s/", board.Name, query.String()),
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
		//updatePipe.Error(err)
		return
	}

	for _, post := range posts {
		offset := int64(post.Num)
		if offset <= prevOffset {
			continue
		}

		update := &service.GenericUpdate{
			Text: html.NewBuilder(maxCaptionSize, 1).
				B().Text(post.DateString).EndB().Br().
				Link(post.URL(), "[LINK]").Br().
				Text("---").Br().
				Parse(post.Comment).
				Build()[0],
		}

		for _, file := range post.Files {
			if file.Type == dvach.WEBM {
				continue
			}

			resource := svc.fs.NewTempResource()
			err := svc.dvach.DownloadFile(file, resource)
			if err == nil {
				if file.Type == dvach.GIF || file.Type == dvach.MP4 {
					update.Type = service.VideoUpdate
				} else {
					update.Type = service.PhotoUpdate
				}

				update.Entity = resource
				break
			} else {
				resource = ""
			}
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
