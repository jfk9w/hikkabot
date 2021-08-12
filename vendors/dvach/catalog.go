package dvach

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/pkg/errors"
)

type CatalogFeedData struct {
	Board  string        `json:"board"`
	Query  *common.Query `json:"query"`
	Offset int           `json:"offset,omitempty"`
	Auto   []string      `json:"auto,omitempty"`
}

func (d *CatalogFeedData) Log() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"board": d.Board,
		"query": d.Query,
		"auto":  d.Auto,
	})
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

func (f *CatalogFeed) ParseSub(ctx context.Context, ref string, options []string) (feed.SubDraft, error) {
	groups := CatalogFeedRefRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return feed.SubDraft{}, feed.ErrWrongVendor
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
				return feed.SubDraft{}, errors.Wrap(err, "compile regexp")
			} else {
				data.Query = &common.Query{Regexp: re}
			}
		}
	}

	catalog, err := f.getCatalog(ctx, data.Board)
	if err != nil {
		return feed.SubDraft{}, errors.Wrap(err, "get catalog")
	}

	draft := feed.SubDraft{
		ID:   data.Board + "/" + data.Query.String(),
		Name: catalog.BoardName + " /" + data.Query.String() + "/",
		Data: data,
	}

	if len(data.Auto) != 0 {
		auto := strings.Join(data.Auto, " ")
		draft.ID += "/" + auto
		draft.Name += " [" + auto + "]"
	}

	return draft, nil
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
		if err, ok := err.(fluhttp.StatusCodeError); ok && err.StatusCode == http.StatusNotFound {
			return errors.Wrap(err, "get catalog")
		}

		data.Log().Warnf("failed to get: %s", err)
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

func (f *CatalogFeed) LoadSub(ctx context.Context, data feed.Data, queue feed.Queue) {
	defer queue.Close()
	if err := f.doLoad(ctx, data, queue); err != nil {
		_ = queue.Submit(ctx, feed.Update{Error: err})
	}
}
