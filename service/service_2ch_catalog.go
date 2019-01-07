package service

import (
	"regexp"
	"sort"
	"strings"

	"github.com/jfk9w-go/hikkabot/api/dvach"
	"github.com/jfk9w-go/hikkabot/common"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type DvachCatalogOptions struct {
	BoardID string `json:"board_id"`
	Query   string `json:"query,omitempty"`
}

type DvachCatalogService struct {
	BaseSubscribeService
	c *dvach.Client
}

func DvachCatalog(baseSubscribe BaseSubscribeService, c *dvach.Client) *DvachCatalogService {
	baseSubscribe.Type = DvachCatalogType
	return &DvachCatalogService{baseSubscribe, c}
}

func (svc *DvachCatalogService) search(boardID string, query *regexp.Regexp) ([]*dvach.Post, error) {
	catalog, err := svc.c.GetCatalog(boardID)
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

var dvachCatalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (svc *DvachCatalogService) Search(input string, query *regexp.Regexp, limit int) ([]common.Envelope, error) {
	groups := dvachCatalogRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return nil, ErrInvalidFormat
	}

	boardId := groups[4]
	posts, err := svc.search(boardId, query)
	if err != nil {
		return nil, err
	}

	envs := make([]common.Envelope, 0)
	for range posts {
		//envs = append(envs, TextEnvelope(post.Comment))
	}

	return envs, nil
}

func (svc *DvachCatalogService) Subscribe(input string, chatId telegram.ID, options string) error {
	groups := dvachCatalogRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return ErrInvalidFormat
	}

	boardId := groups[4]
	query, err := regexp.Compile(options)
	if err != nil {
		return err
	}

	board, err := svc.c.GetBoard(boardId)
	if err != nil {
		return err
	}

	return svc.subscribe(chatId, board.Name, boardId+"/"+query.String(), &DvachCatalogOptions{
		BoardID: board.ID,
		Query:   query.String(),
	})
}

var maxHtmlChunkSize = telegram.MaxMessageSize * 3 / 5

func (svc *DvachCatalogService) Update(currentOffset Offset, rawOptions RawOptions, feed *Feed) {
	defer feed.CloseIn()

	options := new(DvachCatalogOptions)
	err := svc.readOptions(rawOptions, options)
	if err != nil {
		feed.Error(err)
		return
	}

	query := regexp.MustCompile(options.Query)
	posts, err := svc.search(options.BoardID, query)
	if err != nil {
		feed.Error(err)
		return
	}

	for _, post := range posts {
		offset := Offset(post.Num)
		if offset <= currentOffset {
			continue
		}

		text := common.NewHtmlBuilder(maxHtmlChunkSize, 1).
			B().Text(post.DateString).B_().Br().
			Link(post.URL(), "[LINK]").Br().
			Text("---").Br().
			Parse(post.Comment).
			Done("", "")[0]

		if !feed.SubmitText(text, true, offset) {
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
