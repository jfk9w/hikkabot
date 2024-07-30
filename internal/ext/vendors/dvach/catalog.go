package dvach

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w/hikkabot/v4/internal/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/v4/internal/core"
	"github.com/jfk9w/hikkabot/v4/internal/ext/vendors/dvach/internal"
	"github.com/jfk9w/hikkabot/v4/internal/feed"
	"github.com/jfk9w/hikkabot/v4/internal/util"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/telegram-bot-api"
	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

var catalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

type CatalogData struct {
	Board  string      `json:"board"`
	Query  util.Regexp `json:"query,omitempty"`
	Offset int         `json:"offset,omitempty"`
	Auto   []string    `json:"auto,omitempty"`
}

type Catalog[C Context] struct {
	client   dvach.Interface
	mediator feed.Mediator
}

func (v *Catalog[C]) String() string {
	return "2ch/catalog"
}

func (v *Catalog[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
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

func (v *Catalog[C]) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := catalogRegexp.FindStringSubmatch(ref)
	if len(groups) < 6 {
		return nil, nil
	}

	data := &CatalogData{Board: groups[4]}
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
				data.Query.Regexp = re
			}
		}
	}

	catalog, err := v.client.GetCatalog(ctx, data.Board)
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

func (v *Catalog[C]) Refresh(ctx context.Context, header feed.Header, refresh feed.Refresh) error {
	var data CatalogData
	if err := refresh.Init(ctx, &data); err != nil {
		return err
	}

	catalog, err := v.client.GetCatalog(ctx, data.Board)
	if err != nil {
		logf.Get(v).Warnf(ctx, "failed to get catalog for [%s]: %v", header, err)
		return nil
	}

	sort.Sort(internal.Posts(catalog.Threads))
	for i := range catalog.Threads {
		post := &catalog.Threads[i]
		writeHTML := v.writeHTML(ctx, data, post)
		if writeHTML == nil {
			continue
		}

		data.Offset = post.Num
		if err := refresh.Submit(ctx, writeHTML, data); err != nil {
			return err
		}
	}

	return nil
}

func (v *Catalog[C]) writeHTML(ctx context.Context, data CatalogData, post *dvach.Post) feed.WriteHTML {
	if post.Num <= data.Offset {
		return nil
	}

	if !data.Query.MatchString(strings.ToLower(post.Comment)) {
		return nil
	}

	var mediaRef receiver.MediaRef
	if len(post.Files) > 0 {
		mediaRef = v.mediator.Mediate(ctx, post.Files[0].URL(), nil)
	}

	return func(html *tghtml.Writer) error {
		ctx := html.Context()
		if mediaRef != nil {
			ctx = output.With(ctx, tghtml.DefaultMaxCaptionSize, 1)
		}

		if len(data.Auto) != 0 {
			button := (&telegram.Command{Key: "/sub " + post.URL(), Args: data.Auto}).Button("")
			button[0] = button[2]
			ctx = receiver.ReplyMarkup(ctx, telegram.InlineKeyboard([]telegram.Button{button}))
		}

		html = html.WithContext(ctx)

		html.Bold(post.DateString).Text("\n").
			Link("[link]", post.URL())

		if post.Comment != "" {
			html.Text("\n---\n").MarkupString(post.Comment)
		}

		if mediaRef != nil {
			html.Media(post.URL(), mediaRef, true, true)
		}

		return nil
	}
}
