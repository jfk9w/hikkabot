package dvach

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/format"
	_media "github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
)

type ThreadItem struct {
	Board     string
	Num       int
	MediaOnly bool
	Offset    int
	Tag       string
}

type ThreadSource struct {
	*dvach.Client
	*_media.Tor
}

var threadre = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (ThreadSource) ID() string {
	return "dt"
}

func (ThreadSource) Name() string {
	return "Dvach/Thread"
}

func (s ThreadSource) Draft(command, options string, rawData feed.RawData) (*feed.Draft, error) {
	groups := threadre.FindStringSubmatch(command)
	if len(groups) < 6 {
		return nil, feed.ErrDraftFailed
	}

	item := ThreadItem{}
	item.Board = groups[4]
	item.Num, _ = strconv.Atoi(groups[5])
	if strings.HasPrefix(options, "m") {
		item.MediaOnly = true
	} else if strings.HasPrefix(options, "#") {
		item.Tag = options
	}

	post, err := s.GetPost(item.Board, item.Num)
	if err != nil {
		return nil, errors.Wrap(err, "get post")
	}
	rawData.Marshal(item)
	return &feed.Draft{
		ID:   fmt.Sprintf("%s/%d", item.Board, item.Num),
		Name: title(post),
	}, nil
}

func (s ThreadSource) Pull(pull *feed.UpdatePull) error {
	item := new(ThreadItem)
	pull.RawData.Unmarshal(item)
	if item.Offset > 0 {
		item.Offset++
	}

	posts, err := s.GetThread(item.Board, item.Num, item.Offset)
	if err != nil {
		if err, ok := err.(*dvach.Error); ok && err.Code == -http.StatusNotFound {
			return errors.Wrap(err, "get thread")
		} else {
			log.Printf("Failed to load 2ch thread for %s: %s", pull.ID, err)
			return nil
		}
	}

	for _, post := range posts {
		if item.MediaOnly && len(post.Files) == 0 {
			continue
		}

		media := make([]*_media.Promise, len(post.Files))
		for i, file := range post.Files {
			media[i] = s.Submit(file.URL(),
				&mediaDescriptor{s.Client.Client, file},
				_media.Options{
					Hashable: item.MediaOnly,
					Buffer:   true,
				})
		}

		text := format.Text{
			ParseMode: telegram.HTML,
		}

		if !item.MediaOnly {
			b := format.NewHTML(telegram.MaxMessageSize, 0, DefaultSupportedTags, Board(post.Board))
			if item.Tag != "" {
				b.Text(item.Tag)
			} else {
				b.Text(pull.Name)
			}
			b.NewLine().Text(fmt.Sprintf(`#%s%d`, strings.ToUpper(post.Board), post.Num))
			if post.IsOriginal() {
				b.Text(" #OP")
			}
			if post.Comment != "" {
				b.NewLine().
					Text("---").NewLine().
					Parse(post.Comment)
			}
			text = b.Format()
		}

		item.Offset = post.Num
		pull.RawData.Marshal(item)
		update := feed.Update{
			RawData: pull.RawData.Bytes(),
			Pages:   text.Pages,
			Media:   media,
		}

		if !pull.Submit(update) {
			break
		}
	}
	return nil
}

var (
	tagre  = regexp.MustCompile(`<.*?>`)
	junkre = regexp.MustCompile(`(?i)[^\wа-яё]`)
)

func title(post *dvach.Post) string {
	title := html.UnescapeString(post.Subject)
	title = tagre.ReplaceAllString(title, "")
	fields := strings.Fields(title)
	for i, field := range fields {
		fields[i] = strings.Title(junkre.ReplaceAllString(field, ""))
	}
	title = strings.Join(fields, "")
	utf8str := utf8string.NewString(title)
	if utf8str.RuneCount() > 25 {
		return "#" + utf8str.Slice(0, 25)
	}
	return "#" + utf8str.String()
}
