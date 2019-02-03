package dvach

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/html"
	"github.com/jfk9w/hikkabot/service"
)

type catalogOptions struct {
	BoardID string `json:"board_id"`
	Query   string `json:"query,omitempty"`
}

type CatalogService Service

func (s *CatalogService) base() *Service {
	return (*Service)(s)
}

func (s *CatalogService) ID() service.ID {
	return "2ch/catalog"
}

func (s *CatalogService) search(boardID string, query *regexp.Regexp) ([]*dvach.Post, error) {
	catalog, err := s.dvach.GetCatalog(boardID)
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

	sort.Sort(catalogPosts(result))
	return result, nil
}

var catalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (s *CatalogService) Subscribe(input string, chat *service.EnrichedChat, options string) error {
	groups := catalogRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return service.ErrInvalidFormat
	}

	boardID := groups[4]
	query, err := regexp.Compile(options)
	if err != nil {
		return err
	}

	board, err := s.dvach.GetBoard(boardID)
	if err != nil {
		return err
	}

	return s.agg.Subscribe(chat,
		s.ID(),
		fmt.Sprintf("%s/%s", boardID, query),
		fmt.Sprintf("%s /%s/", board.Name, query.String()),
		&catalogOptions{
			BoardID: board.ID,
			Query:   query.String(),
		})
}

func (s *CatalogService) Update(prevOffset int64, optionsFunc service.OptionsFunc, pipe *service.UpdatePipe) {
	defer pipe.Close()
	options := new(catalogOptions)
	err := optionsFunc(options)
	if err != nil {
		pipe.Err = err
		return
	}

	query := regexp.MustCompile(options.Query)
	posts, err := s.search(options.BoardID, query)
	if err != nil {
		//pipe.Err = err
		return
	}

	for _, post := range posts {
		offset := int64(post.Num)
		if offset <= prevOffset {
			continue
		}

		var mediaOut <-chan service.MediaResponse
		for _, file := range post.Files {
			mediaOut = s.base().download(file)
			break
		}

		update := service.Update{
			Offset: offset,
			Text: service.UpdateTextSlice(html.NewBuilder(service.MaxCaptionSize, 1).
				B().Text(post.DateString).EndB().Br().
				Link(post.URL(), "[LINK]").Br().
				Text("---").Br().
				Parse(post.Comment).
				Build()),
			MediaSize: minInt(len(post.Files), 1),
			Media:     mediaOut,
		}

		if !pipe.Submit(update) {
			return
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

type catalogPosts []*dvach.Post

func (p catalogPosts) Len() int {
	return len(p)
}

func (p catalogPosts) Less(i, j int) bool {
	return p[i].Num < p[j].Num
}

func (p catalogPosts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
