package dvach

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/app/media"
	"github.com/jfk9w/hikkabot/app/subscription"
	"github.com/jfk9w/hikkabot/html"
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
	return "2ch catalog"
}

func (c *Catalog) ID() string {
	return c.Board + "/" + c.Query.String()
}

func (c *Catalog) Name() string {
	return fmt.Sprintf("/%s/%s/", c.Board, c.Query.String())
}

var catalogRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)(/)?$`)

func (c *Catalog) Parse(ctx subscription.Context, cmd string, opts string) error {
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
		var me []media.Media
		for _, file := range thread.Files {
			me = []media.Media{createMedia(ctx, &file)}
			ctx.MediaManager.Download(me)
			break
		}

		update := subscription.Update{
			Offset: subscription.Offset(thread.Num),
			Text: html.NewBuilder(telegram.MaxCaptionSize, 1).
				B().Text(thread.DateString).EndB().Br().
				Link(thread.URL(), "[link]").Br().
				Text("---").Br().
				Parse(comment(thread.Comment)).
				Build(),
			Media: me,
		}

		select {
		case <-uc.Cancel():
			return
		case uc.C <- update:
			continue
		}
	}
}
