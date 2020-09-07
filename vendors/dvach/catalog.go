package dvach

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/pkg/errors"
)

type CatalogFeedData struct {
	Board  string        `json:"board"`
	Query  *common.Query `json:"query"`
	Offset int           `json:"offset,omitempty"`
	Auto   []string      `json:"auto,omitempty"`
}

type CatalogFeed struct {
	*Client
	*feed.MediaManager
}

func (f *CatalogFeed) getCatalog(ctx context.Context, board string) (*Catalog, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return f.Client.GetCatalog(ctx, board)
}

var CatalogFeedRefRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (f *CatalogFeed) Parse(ctx context.Context, ref string, options []string) (feed.Candidate, error) {
	groups := CatalogFeedRefRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return feed.Candidate{}, feed.ErrWrongVendor
	}

	data := CatalogFeedData{Board: groups[4]}
loop:
	for i, option := range options {
		switch {
		case option == "auto":
			data.Auto = options[i+1:]
			break loop

		case strings.HasPrefix(option, "re="):
			option = option[3:]
			fallthrough
		default:
			if re, err := regexp.Compile(option); err != nil {
				return feed.Candidate{}, errors.Wrap(err, "compile regexp")
			} else {
				data.Query = &common.Query{Regexp: re}
			}
		}
	}

	catalog, err := f.getCatalog(ctx, data.Board)
	if err != nil {
		return feed.Candidate{}, errors.Wrap(err, "get catalog")
	}

	candidate := feed.Candidate{
		ID:   data.Board + "/" + data.Query.String(),
		Name: catalog.BoardName + " /" + data.Query.String() + "/",
		Data: data,
	}

	if len(data.Auto) != 0 {
		auto := strings.Join(data.Auto, " ")
		candidate.ID += "/" + auto
		candidate.Name += " [" + auto + "]"
	}

	return candidate, nil
}

type catalogFeedQueryResult []Post

func (r catalogFeedQueryResult) Len() int {
	return len(r)
}

func (r catalogFeedQueryResult) Less(i, j int) bool {
	return r[i].Num < r[j].Num
}

func (r catalogFeedQueryResult) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (f *CatalogFeed) doLoad(ctx context.Context, rawData feed.Data, queue feed.Queue) error {
	data := new(CatalogFeedData)
	if err := rawData.ReadTo(data); err != nil {
		return errors.Wrap(err, "read data")
	}

	catalog, err := f.getCatalog(ctx, data.Board)
	if err != nil {
		if err, ok := err.(fluhttp.StatusCodeError); ok && err.Code == http.StatusNotFound {
			return errors.Wrap(err, "get catalog")
		}

		log.Printf("[dvach > catalog > /%s /%s/] failed to get: %s", data.Board, data.Query.String(), err)
		return nil
	}

	sort.Sort(catalogFeedQueryResult(catalog.Threads))
	for _, post := range catalog.Threads {
		post := post
		if post.Num <= data.Offset {
			continue
		}

		if !data.Query.MatchString(strings.ToLower(post.Comment)) {
			continue
		}

		var media format.MediaRef = nil
		if len(post.Files) > 0 {
			media = f.MediaManager.Submit(newMediaRef(f.Client.Client, queue.SubID.FeedID, post.Files[0], false))
		}

		write := func(html *format.HTMLWriter) error {
			if media != nil {
				html.Session.PageSize = format.DefaultMaxCaptionSize
				html.Session.PageCount = 1
			}

			ctx := html.Session.Context
			if len(data.Auto) != 0 {
				button := telegram.Command{Key: "/sub " + post.URL(), Args: data.Auto}.Button("")
				button[0] = button[2]
				html.Session.Context = format.WithReplyMarkup(ctx, telegram.InlineKeyboard([]telegram.Button{button}))
			}

			html.Bold(post.DateString).Text("\n").
				Link("[link]", post.URL())

			if post.Comment != "" {
				html.Text("\n---\n").MarkupString(post.Comment)
			}

			if media != nil {
				html.Media(post.URL(), media, true)
			}

			return nil
		}

		data.Offset = post.Num
		if err := queue.Submit(ctx, feed.Update{
			Write: write,
			Data:  *data,
		}); err != nil {
			return nil
		}
	}

	return nil
}

func (f *CatalogFeed) Load(ctx context.Context, data feed.Data, queue feed.Queue) {
	defer queue.Close()
	if err := f.doLoad(ctx, data, queue); err != nil {
		_ = queue.Submit(ctx, feed.Update{Error: err})
	}
}
