package dvach

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/subscription"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
)

func ThreadService() subscription.Item {
	return new(Thread)
}

type Thread struct {
	Board     string
	Num       int
	Title     string
	MediaOnly bool
}

func (t *Thread) Service() string {
	return "Dvach/Thread"
}

func (t *Thread) ID() string {
	return fmt.Sprintf("%s/%d", t.Board, t.Num)
}

func (t *Thread) Name() string {
	return t.Title
}

var threadRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (t *Thread) Parse(ctx subscription.ApplicationContext, cmd string, opts string) error {
	groups := threadRegexp.FindStringSubmatch(cmd)
	if len(groups) < 6 {
		return subscription.ErrParseFailed
	}
	board := groups[4]
	num, _ := strconv.Atoi(groups[5])
	mediaOnly := false
	if strings.HasPrefix(opts, "m") {
		mediaOnly = true
	}
	post, err := ctx.DvachClient.GetPost(board, num)
	if err != nil {
		return errors.Wrap(err, "on post load")
	}
	t.Board = board
	t.Num = num
	t.Title = threadTitle(post)
	t.MediaOnly = mediaOnly
	return nil
}

func (t *Thread) Update(ctx subscription.ApplicationContext, offset int64, queue *subscription.UpdateQueue) {
	if offset > 0 {
		offset++
	}
	posts, err := ctx.DvachClient.GetThread(t.Board, t.Num, int(offset))
	if err != nil {
		queue.Fail(errors.Wrap(err, "on posts load"))
		return
	}
	for _, post := range posts {
		if t.MediaOnly && len(post.Files) == 0 {
			continue
		}
		media := make([]*media.Media, len(post.Files))
		for i, file := range post.Files {
			media[i] = downloadMedia(ctx, file)
		}
		text := format.NewHTML(telegram.MaxMessageSize, 0, DefaultSupportedTags, Board(post.Board)).
			Text(t.Title).NewLine().
			Text(fmt.Sprintf(`#%s%d`, strings.ToUpper(post.Board), post.Num))
		if post.IsOriginal() {
			text.Text(" #OP")
		}
		if !t.MediaOnly && post.Comment != "" {
			text.NewLine().
				Text("---").NewLine().
				Parse(post.Comment)
		}
		update := subscription.Update{
			Offset: int64(post.Num),
			Text:   text.Format(),
			Media:  media,
		}
		if !queue.Offer(update) {
			return
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
		return "#" + utf8str.Slice(0, 25)
	}
	return "#" + utf8str.String()
}
