package dvach

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/app/media"
	"github.com/jfk9w/hikkabot/app/subscription"
	"github.com/jfk9w/hikkabot/html"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
)

func ThreadFactory() subscription.Interface {
	return new(Thread)
}

type Thread struct {
	Board     string
	Num       int
	Title     string
	MediaOnly bool
}

func (t *Thread) Service() string {
	return "2ch thread"
}

var threadRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (t *Thread) Parse(ctx subscription.Context, cmd string, opts string) (string, error) {
	groups := threadRegexp.FindStringSubmatch(cmd)
	if len(groups) < 6 {
		return subscription.EmptyHash, subscription.ErrParseFailed
	}

	board := groups[4]
	num, _ := strconv.Atoi(groups[5])
	mediaOnly := false
	if strings.HasPrefix(opts, "m") {
		mediaOnly = true
	}

	post, err := ctx.DvachClient.GetPost(board, num)
	if err != nil {
		return subscription.EmptyHash, errors.Wrap(err, "on post load")
	}

	t.Board = board
	t.Num = num
	t.Title = threadTitle(post)
	t.MediaOnly = mediaOnly

	return fmt.Sprintf("%s/%d", board, num), nil
}

func (t *Thread) Update(ctx subscription.Context, offset subscription.Offset, uc *subscription.UpdateCollection) {
	defer close(uc.C)
	if offset > 0 {
		offset++
	}

	posts, err := ctx.DvachClient.GetThread(t.Board, t.Num, int(offset))
	if err != nil {
		uc.Err = errors.Wrap(err, "on posts load")
		return
	}

	for _, post := range posts {
		me := make([]media.Media, len(post.Files))
		for i := range post.Files {
			me[i] = createMedia(ctx, &post.Files[i])
		}
		ctx.MediaManager.Download(me)

		b := html.NewBuilder(telegram.MaxMessageSize, -1).
			Text(`#` + t.Title).Br().
			Text(fmt.Sprintf(`#%s%d`, strings.ToUpper(post.Board), post.Num))
		if post.IsOriginal() {
			b.Text(" #OP")
		}
		if !t.MediaOnly && post.Comment != "" {
			b.Br().
				Text("---").Br().
				Parse(comment(post.Comment))
		}

		update := subscription.Update{
			Offset: int64(post.Num),
			Text:   b.Build(),
			Media:  me,
		}

		select {
		case <-uc.Interrupt():
			return
		case uc.C <- update:
			continue
		}
	}
}

var (
	tagRegexp  = regexp.MustCompile(`<.*?>`)
	junkRegexp = regexp.MustCompile(`(?i)[^\wа-яё]`)
)

func threadTitle(post *dvach.Post) string {
	title := html.UnescapeString(post.Subject)
	title = tagRegexp.ReplaceAllString(title, "")
	fields := strings.Fields(title)

	for i, field := range fields {
		fields[i] = strings.Title(junkRegexp.ReplaceAllString(field, ""))
	}

	title = strings.Join(fields, "")
	utf8str := utf8string.NewString(title)
	if utf8str.RuneCount() > 25 {
		return utf8str.Slice(0, 25)
	}

	return utf8str.String()
}
