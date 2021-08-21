package catalog

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/vendors"
	dvachVendor "github.com/jfk9w/hikkabot/vendors/dvach"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Vendor struct {
	DvachClient  *dvach.Client
	MediaManager *feed.MediaManager
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
		_ = queue.Cancel(ctx, err)
		return
	}

	sort.Sort(threadSorter(catalog.Threads))
	for i := range catalog.Threads {
		post := &catalog.Threads[i]
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
			log.Warnf("interrupted refresh: %s", err)
			return
		}
	}
}

func (v *Vendor) processPost(
	header *feed.Header, data *Data,
	log *logrus.Entry, post *dvach.Post) (
	writeHTML feed.WriteHTML, err error) {

	log = log.WithField("num", post.Num)

	if post.Num <= data.Offset {
		log.Tracef("skip: num < data offset %d", data.Offset)
		return nil, nil
	}

	if !data.Query.MatchString(strings.ToLower(post.Comment)) {
		log.Tracef("skip: does not match pattern (%s)", data.Query)
		return nil, nil
	}

	var media richtext.MediaRef = nil
	if len(post.Files) > 0 {
		media = v.MediaManager.Submit(dvachVendor.NewMediaRef(v.DvachClient.Unmask(), header.FeedID, post.Files[0], false))
	}

	return func(html *richtext.HTMLWriter) error {
		if media != nil {
			html.Session.PageSize = richtext.DefaultMaxCaptionSize
			html.Session.PageCount = 1
		}

		ctx := html.Session.Context
		if len(data.Auto) != 0 {
			button := telegram.Command{Key: "/sub " + post.URL(), Args: data.Auto}.Button("")
			button[0] = button[2]
			html.Session.Context = richtext.WithReplyMarkup(ctx, telegram.InlineKeyboard([]telegram.Button{button}))
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
	}, nil
}

func (v *Vendor) getCatalog(ctx context.Context, board string) (*dvach.Catalog, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return v.DvachClient.GetCatalog(ctx, board)
}
