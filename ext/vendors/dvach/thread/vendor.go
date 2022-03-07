package thread

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hikkabot/3rdparty/dvach"
	"hikkabot/core/feed"
	"hikkabot/core/media"
	"hikkabot/ext/vendors"
	dvachVendor "hikkabot/ext/vendors/dvach"

	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	tgmedia "github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Vendor struct {
	DvachClient  *dvach.Client
	MediaManager *media.Manager
	GetTimeout   time.Duration
}

var refRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (v *Vendor) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := refRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return nil, feed.ErrWrongVendor
	}

	data := &Data{Board: groups[4]}
	data.Num, _ = strconv.Atoi(groups[5])
	for _, option := range options {
		switch {
		case option == "m":
			data.MediaOnly = true
		case strings.HasPrefix(option, "#"):
			data.Tag = option
		}
	}

	post, err := v.getPost(ctx, data.Board, data.Num)
	if err != nil {
		return nil, errors.Wrap(err, "get post")
	}

	if data.Tag == "" {
		data.Tag = vendors.Hashtag(post.Subject)
	}

	return &feed.Draft{
		SubID: fmt.Sprintf("%s/%d", data.Board, data.Num),
		Name:  data.Tag,
		Data:  &data,
	}, nil
}

func (v *Vendor) Refresh(ctx context.Context, queue *feed.Queue) {
	data := new(Data)
	if err := queue.GetData(ctx, data); err != nil {
		return
	}
	log := queue.Log(ctx, data)
	posts, err := v.DvachClient.GetThread(ctx, data.Board, data.Num, data.Offset)
	if err != nil {
		if derr := new(dvach.Error); errors.As(err, &derr) && derr.Code == -http.StatusNotFound {
			_ = queue.Cancel(ctx, err)
		} else {
			log.Warnf("update: failed (%v)", err)
		}

		return
	}

	for i := range posts {
		post := &posts[i]
		log := log.WithField("num", post.Num)
		writeHTML, err := v.processPost(queue.Header, data, log, post)
		if err != nil {
			_ = queue.Cancel(ctx, err)
			return
		}

		if writeHTML == nil {
			continue
		}

		data.Offset = post.Num + 1
		if err := queue.Proceed(ctx, writeHTML, data); err != nil {
			return
		}
	}
}

func (v *Vendor) processPost(
	header *feed.Header, data *Data,
	log *logrus.Entry, post *dvach.Post) (
	writeHTML feed.WriteHTML, err error) {

	if data.MediaOnly && len(post.Files) == 0 {
		log.Debug("update: skip (media only)")
		return nil, nil
	}

	media := make([]tgmedia.Ref, len(post.Files))
	for i, file := range post.Files {
		ref := dvachVendor.NewMediaRef(v.DvachClient.Unmask(), header.FeedID, file, data.MediaOnly)
		media[i] = v.MediaManager.Submit(ref)
	}

	return func(html *html.Writer) error {
		if !data.MediaOnly {
			writePost(html, post, data.Tag)
			for i, media := range media {
				html.Media(post.Files[i].URL(), media, len(post.Files) == 1, true)
			}
		} else {
			if output, ok := html.Out.(*output.Paged); ok {
				if chat, ok := output.Receiver.(*receiver.Chat); ok {
					chat.SkipOnMediaError = true
				}
			}

			for i, media := range media {
				html.Text(data.Tag).Media(post.Files[i].URL(), media, true, true)
			}
		}

		return nil
	}, nil
}

func (v *Vendor) getPost(ctx context.Context, board string, num int) (*dvach.Post, error) {
	ctx, cancel := context.WithTimeout(ctx, v.GetTimeout)
	defer cancel()
	return v.DvachClient.GetPost(ctx, board, num)
}

func writePost(html *html.Writer, post *dvach.Post, tag string) {
	if tag == "" {
		tag = vendors.Hashtag(post.Subject)
	}

	html.Anchors = anchorFormat{post.Board}
	html.Text(tag).Text(fmt.Sprintf("\n#%s%d", strings.ToUpper(post.Board), post.Num))
	if post.IsOriginal() {
		html.Text(" #OP")
	}

	if post.Comment != "" {
		html.Text("\n---\n").MarkupString(post.Comment)
	}
}
