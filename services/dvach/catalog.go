package dvach

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/subscription"
	"github.com/pkg/errors"
)

func CatalogService() subscription.Item {
	return new(Catalog)
}

type Catalog struct {
	Board string
	Query query
}

func (c *Catalog) Service() string {
	return "Dvach/Catalog"
}

func (c *Catalog) ID() string {
	return c.Board + "/" + c.Query.String()
}

func (c *Catalog) Name() string {
	return fmt.Sprintf("/%s/%s/", c.Board, c.Query.String())
}

var catalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (c *Catalog) Parse(_ subscription.Context, cmd string, opts string) error {
	groups := catalogRegexp.FindStringSubmatch(cmd)
	if len(groups) < 6 {
		return subscription.ErrParseFailed
	}
	board := groups[4]
	var re *regexp.Regexp
	if opts != "" {
		var err error
		re, err = regexp.Compile(opts)
		if err != nil {
			return errors.Wrap(err, "on regexp compilation")
		}
	}
	c.Board = board
	c.Query = query{re}
	return nil
}

func (c *Catalog) Update(ctx subscription.Context, offset subscription.Offset, uc *subscription.UpdateCollection) {
	defer close(uc.C)
	catalog, err := ctx.DvachClient.GetCatalog(c.Board)
	if err != nil {
		uc.Error = errors.Wrap(err, "on catalog load")
		return
	}
	results := make([]dvach.Post, 0)
	for _, thread := range catalog.Threads {
		matches := thread.Num > int(offset)
		matches = matches && (c.Query.Regexp == nil || c.Query.MatchString(strings.ToLower(thread.Comment)))
		if matches {
			results = append(results, thread)
		}
	}
	sort.Sort(queryResults(results))
	for _, thread := range results {
		var mediaBatch media.Batch
		for i := range thread.Files {
			file := &thread.Files[i]
			mediaBatch = media.NewBatch(defaultMediaLoader{file, ctx.DvachClient})
			ctx.MediaManager.Download(mediaBatch)
			break
		}
		update := subscription.Update{
			Offset: subscription.Offset(thread.Num),
			Text: format.NewHTML(telegram.MaxCaptionSize, 1, DefaultSupportedTags, Board(thread.Board)).
				Tag("b").Text(thread.DateString).EndTag().NewLine().
				Link(thread.URL(), "[link]").NewLine().
				Text("---").NewLine().
				Parse(thread.Comment).
				Pages(),
			Media: mediaBatch,
		}
		select {
		case <-uc.Cancel():
			return
		case uc.C <- update:
			continue
		}
	}
}
