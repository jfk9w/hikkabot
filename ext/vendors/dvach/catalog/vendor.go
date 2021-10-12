package catalog

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	tgmedia "github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"hikkabot/3rdparty/dvach"
	"hikkabot/core/feed"
	"hikkabot/core/media"
	"hikkabot/ext/vendors"
	dvachVendor "hikkabot/ext/vendors/dvach"
)

type Vendor struct {
	DvachClient  *dvach.Client
	MediaManager *media.Manager
	GetTimeout   time.Duration
}

var refRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (v *Vendor) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := refRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return nil, feed.ErrWrongVendor
	}

	data := &Data{Board: groups[4]}
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
				return nil, errors.Wrap(err, "compile regexp")
			} else {
				data.Query = &vendors.Query{Regexp: re}
			}
		}
	}

	catalog, err := v.getCatalog(ctx, data.Board)
	if err != nil {
		return nil, errors.Wrap(err, "get catalog")
	}

	draft := &feed.Draft{
		SubID: data.Board + "/" + data.Query.String(),
		Name:  catalog.BoardName + " /" + data.Query.String() + "/",
		Data:  &data,
	}

	if len(data.Auto) != 0 {
		auto := strings.Join(data.Auto, " ")
		draft.SubID += "/" + auto
		draft.Name += " [" + auto + "]"
	}

	return draft, nil
}

func (v *Vendor) Refresh(ctx context.Context, queue *feed.Queue) {
	data := new(Data)
	if err := queue.GetData(ctx, data); err != nil {
		return
	}

	log := queue.Log(ctx, data)
	catalog, err := v.DvachClient.GetCatalog(ctx, data.Board)
	if err != nil {
		log.Warnf("update: failed")
		return
	}

	sort.Sort(threadSorter(catalog.Threads))
	for i := range catalog.Threads {
		post := &catalog.Threads[i]
		log := log.WithField("num", post.Num)
		writeHTML, err := v.processPost(queue.Header, data, log, post)
		if err != nil {
			_ = queue.Cancel(ctx, err)
			return
		}

		if writeHTML == nil {
			continue
		}

		data.Offset = post.Num
		if err := queue.Proceed(ctx, writeHTML, data); err != nil {
			return
		}
	}
}

func (v *Vendor) processPost(
	header *feed.Header, data *Data,
	log *logrus.Entry, post *dvach.Post) (
	writeHTML feed.WriteHTML, err error) {

	if post.Num <= data.Offset {
		log.Debugf("update: skip (num < data offset %d)", data.Offset)
		return nil, nil
	}

	if !data.Query.MatchString(strings.ToLower(post.Comment)) {
		log.Debug("update: skip (pattern mismatch)")
		return nil, nil
	}

	var media tgmedia.Ref = nil
	if len(post.Files) > 0 {
		media = v.MediaManager.Submit(dvachVendor.NewMediaRef(v.DvachClient.Unmask(), header.FeedID, post.Files[0], false))
	}

	return func(html *html.Writer) error {
		if media != nil {
			if out, ok := html.Out.(*output.Paged); ok {
				out.PageSize = tghtml.DefaultMaxCaptionSize
				out.PageCount = 1
			}
		}

		if len(data.Auto) != 0 {
			if out, ok := html.Out.(*output.Paged); ok {
				if chat, ok := out.Receiver.(*receiver.Chat); ok {
					button := (&telegram.Command{Key: "/sub " + post.URL(), Args: data.Auto}).Button("")
					button[0] = button[2]
					chat.ReplyMarkup = telegram.InlineKeyboard([]telegram.Button{button})
				}
			}
		}

		html.Bold(post.DateString).Text("\n").
			Link("[link]", post.URL())

		if post.Comment != "" {
			html.Text("\n---\n").MarkupString(post.Comment)
		}

		if media != nil {
			html.Media(post.URL(), media, true, true)
		}

		return nil
	}, nil
}

func (v *Vendor) getCatalog(ctx context.Context, board string) (*dvach.Catalog, error) {
	ctx, cancel := context.WithTimeout(ctx, v.GetTimeout)
	defer cancel()
	return v.DvachClient.GetCatalog(ctx, board)
}
