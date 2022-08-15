package dvach

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu/logf"
	"hikkabot/3rdparty/dvach"
	"hikkabot/core"
	"hikkabot/ext/vendors/dvach/internal"
	"hikkabot/feed"
	"hikkabot/util"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

var threadRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

type ThreadData struct {
	Board     string `json:"board"`
	Num       int    `json:"num"`
	MediaOnly bool   `json:"media_only,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Tag       string `json:"tag"`
}

type Thread[C Context] struct {
	client   dvach.Interface
	mediator feed.Mediator
}

func (v Thread[C]) String() string {
	return "2ch/thread"
}

func (v *Thread[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var client dvach.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	var mediator core.Mediator[C]
	if err := app.Use(ctx, &mediator, false); err != nil {
		return err
	}

	v.client = &client
	v.mediator = &mediator
	return nil
}

func (v *Thread[C]) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := threadRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return nil, nil
	}

	data := &ThreadData{Board: groups[4]}
	data.Num, _ = strconv.Atoi(groups[5])
	for _, option := range options {
		switch {
		case option == "m":
			data.MediaOnly = true
		case strings.HasPrefix(option, "#"):
			data.Tag = option
		}
	}

	post, err := v.client.GetPost(ctx, data.Board, data.Num)
	if err != nil {
		return nil, errors.Wrap(err, "get post")
	}

	if data.Tag == "" {
		data.Tag = util.Hashtag(post.Subject)
	}

	return &feed.Draft{
		SubID: fmt.Sprintf("%s/%d", data.Board, data.Num),
		Name:  data.Tag,
		Data:  &data,
	}, nil
}

func (v *Thread[C]) Refresh(ctx context.Context, header feed.Header, refresh feed.Refresh) error {
	var data ThreadData
	if err := refresh.Init(ctx, &data); err != nil {
		return err
	}

	posts, err := v.client.GetThread(ctx, data.Board, data.Num, data.Offset)
	if err != nil {
		var dvachErr dvach.Error
		if errors.As(err, &dvachErr) && dvachErr.Code == -http.StatusNotFound {
			return err
		}

		logf.Get(v).Warnf(ctx, "failed to get posts for [%s]: %v", header, err)
		return nil
	}

	for i := range posts {
		post := &posts[i]
		writeHTML := v.writeHTML(ctx, header, data, post)
		if writeHTML == nil {
			continue
		}

		data.Offset = post.Num + 1
		if err := refresh.Submit(ctx, writeHTML, data); err != nil {
			return err
		}
	}

	return nil
}

func (v *Thread[C]) writeHTML(ctx context.Context, header feed.Header, data ThreadData, post *dvach.Post) feed.WriteHTML {
	if data.MediaOnly && len(post.Files) == 0 {
		return nil
	}

	var dedupKey *feed.ID
	if data.MediaOnly {
		dedupKey = &header.FeedID
	}

	mediaRefs := make([]receiver.MediaRef, len(post.Files))
	for i, file := range post.Files {
		mediaRefs[i] = v.mediator.Mediate(ctx, file.URL(), dedupKey)
	}

	return func(html *html.Writer) error {
		if !data.MediaOnly {
			if data.Tag == "" {
				data.Tag = util.Hashtag(post.Subject)
			}

			html.Anchors = internal.AnchorFormat{Board: post.Board}
			html.Text(data.Tag).Text(fmt.Sprintf("\n#%s%d", strings.ToUpper(post.Board), post.Num))
			if post.IsOriginal() {
				html.Text(" #OP")
			}

			if post.Comment != "" {
				html.Text("\n---\n").MarkupString(post.Comment)
			}

			for i, mediaRef := range mediaRefs {
				html.Media(post.Files[i].URL(), mediaRef, len(post.Files) == 1, true)
			}

			return nil
		}

		html = html.WithContext(receiver.SkipOnMediaError(html.Context()))
		for i, mediaRef := range mediaRefs {
			html.Text(data.Tag).Media(post.Files[i].URL(), mediaRef, true, true)
		}

		return nil
	}
}
