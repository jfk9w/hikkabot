package dvach

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
)

type ThreadFeedData struct {
	Board     string `json:"board"`
	Num       int    `json:"num"`
	MediaOnly bool   `json:"media_only,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Tag       string `json:"tag"`
}

type ThreadFeed struct {
	*Client
	*feed.MediaManager
}

var ThreadFeedRefRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (f *ThreadFeed) getPost(ctx context.Context, board string, num int) (Post, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return f.Client.GetPost(ctx, board, num)
}

func (f *ThreadFeed) getThread(ctx context.Context, board string, num int, offset int) ([]Post, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return f.Client.GetThread(ctx, board, num, offset)
}

var (
	dvachThreadTagRegexp  = regexp.MustCompile(`<.*?>`)
	dvachThreadJunkRegexp = regexp.MustCompile(`(?i)[^\wа-яё]`)
)

func getTitle(post Post) string {
	title := html.UnescapeString(post.Subject)
	title = dvachThreadTagRegexp.ReplaceAllString(title, "")
	fields := strings.Fields(title)
	for i, field := range fields {
		fields[i] = strings.Title(dvachThreadJunkRegexp.ReplaceAllString(field, ""))
	}
	title = strings.Join(fields, "")
	utf8str := utf8string.NewString(title)
	if utf8str.RuneCount() > 25 {
		return "#" + utf8str.Slice(0, 25)
	}
	return "#" + utf8str.String()
}

func (f *ThreadFeed) ParseSub(ctx context.Context, ref string, options []string) (feed.SubDraft, error) {
	groups := ThreadFeedRefRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return feed.SubDraft{}, feed.ErrWrongVendor
	}

	data := ThreadFeedData{Board: groups[4]}
	data.Num, _ = strconv.Atoi(groups[5])
	for _, option := range options {
		switch {
		case option == "m":
			data.MediaOnly = true
		case strings.HasPrefix(option, "#"):
			data.Tag = option
		}
	}

	post, err := f.getPost(ctx, data.Board, data.Num)
	if err != nil {
		return feed.SubDraft{}, errors.Wrap(err, "get post")
	}

	if data.Tag == "" {
		data.Tag = getTitle(post)
	}

	return feed.SubDraft{
		ID:   fmt.Sprintf("%s/%d", data.Board, data.Num),
		Name: data.Tag,
		Data: data,
	}, nil
}

func writePost(html *format.HTMLWriter, post Post, tag string) {
	if tag == "" {
		tag = getTitle(post)
	}

	html.AnchorFormat = anchorFormat{post.Board}
	html.Text(tag).Text(fmt.Sprintf("\n#%s%d", strings.ToUpper(post.Board), post.Num))
	if post.IsOriginal() {
		html.Text(" #OP")
	}

	if post.Comment != "" {
		html.Text("\n---\n").MarkupString(post.Comment)
	}
}

func (f *ThreadFeed) doLoad(ctx context.Context, rawData feed.Data, queue feed.Queue) error {
	data := new(ThreadFeedData)
	if err := rawData.ReadTo(data); err != nil {
		return errors.Wrap(err, "read data")
	}

	posts, err := f.getThread(ctx, data.Board, data.Num, data.Offset)
	if err != nil {
		if err, ok := err.(*Error); ok && err.Code == -http.StatusNotFound {
			return errors.Wrap(err, "get thread")
		}

		log.Printf("[dvach > thread > /%s/%d] failed to get: %s", data.Board, data.Num, err)
		return nil
	}

	for _, post := range posts {
		post := post
		if data.MediaOnly && len(post.Files) == 0 {
			continue
		}

		media := make([]format.MediaRef, len(post.Files))
		for i, file := range post.Files {
			media[i] = f.MediaManager.Submit(
				newMediaRef(f.Client.Client, queue.SubID.FeedID, file, data.MediaOnly))
		}

		write := func(html *format.HTMLWriter) error {
			if !data.MediaOnly {
				writePost(html, post, data.Tag)
				for i, media := range media {
					html.Media(post.Files[i].URL(), media, len(post.Files) == 1)
				}
			} else {
				for i, media := range media {
					html.Text(data.Tag).Media(post.Files[i].URL(), media, true)
				}
			}

			return nil
		}

		data.Offset = post.Num + 1
		if err := queue.Submit(ctx, feed.Update{
			Write: write,
			Data:  *data,
		}); err != nil {
			return nil
		}
	}

	return nil
}

func (f *ThreadFeed) LoadSub(ctx context.Context, data feed.Data, queue feed.Queue) {
	defer queue.Close()
	if err := f.doLoad(ctx, data, queue); err != nil {
		_ = queue.Submit(ctx, feed.Update{Error: err})
	}
}

type anchorFormat struct {
	board string
}

func (f anchorFormat) Format(text string, attrs format.HTMLAttributes) string {
	if dataNum := attrs.Get("data-num"); dataNum != "" {
		return fmt.Sprintf(`#%s%s`, strings.ToUpper(f.board), dataNum)
	} else {
		return format.DefaultHTMLAnchorFormat.Format(text, attrs)
	}
}
