package dvach

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/html"
	"github.com/jfk9w/hikkabot/service"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
)

type threadOptions struct {
	BoardID string `json:"board_id"`
	Num     int    `json:"num"`
	Title   string `json:"title"`
	Mode    string `json:"mode,omitempty"`
}

type ThreadService Service

func (s *ThreadService) base() *Service {
	return (*Service)(s)
}

func (s *ThreadService) ID() service.ID {
	return "2ch/thread"
}

var threadRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (s *ThreadService) Subscribe(input string, chat *service.EnrichedChat, args string) error {
	groups := threadRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return service.ErrInvalidFormat
	}

	boardID := groups[4]
	threadID, _ := strconv.Atoi(groups[5])

	var mode string
	if args != "" {
		if args != "media_only" {
			return errors.Errorf("invalid mode %s", args)
		}

		mode = args
	}

	post, err := s.dvach.GetPost(boardID, threadID)
	if err != nil {
		return err
	}

	title := threadTitle(post)
	return s.agg.Subscribe(chat, s.ID(), post.BoardID+"/"+post.ParentString, title, &threadOptions{
		BoardID: boardID,
		Num:     threadID,
		Title:   title,
		Mode:    mode,
	})
}

type postMessageKey struct {
	boardID  string
	threadID int
	num      int
}

func (k *postMessageKey) String() string {
	return fmt.Sprintf("%s/%d/%d", k.boardID, k.threadID, k.num)
}

func (s *ThreadService) Update(prevOffset int64, optionsFunc service.OptionsFunc, pipe *service.UpdatePipe) {
	defer pipe.Close()
	options := new(threadOptions)
	err := optionsFunc(options)
	if err != nil {
		pipe.Err = err
		return
	}

	if prevOffset > 0 {
		prevOffset += 1
	}

	posts, err := s.dvach.GetThread(options.BoardID, options.Num, int(prevOffset))
	if err != nil {
		pipe.Err = err
		return
	}

	if len(posts) == 0 {
		return
	}

	for _, post := range posts {
		var mediaOut <-chan service.MediaResponse
		if len(post.Files) > 0 {
			mediaOut = s.base().download(post.Files...)
		} else if options.Mode != "" {
			continue
		}

		b := html.NewBuilder(service.MaxMessageSize, -1).
			Text(`#` + options.Title).Br().
			Text(fmt.Sprintf(`#%s%d`, strings.ToUpper(post.BoardID), post.Num))

		if post.IsOriginal() {
			b.Text(" #OP")
		}

		if options.Mode == "" && post.Comment != "" {
			b.Br().
				Text("---").Br().
				Parse(post.Comment)
		}

		update := service.Update{
			Offset:    int64(post.Num),
			Text:      s.updateTextFunc(b.Build()),
			MediaSize: len(post.Files),
			Media:     mediaOut,
			Key: &postMessageKey{
				boardID:  post.BoardID,
				threadID: post.Parent,
				num:      post.Num,
			},
		}

		if !pipe.Submit(update) {
			return
		}
	}
}

var replyRegexp = regexp.MustCompile(`<a\s+href=".*?/([a-zA-Z0-9]+)/res/([0-9]+)\.html#([0-9]+)".*?>.*?</a>`)

func (s *ThreadService) updateTextFunc(text []string) service.UpdateTextFunc {
	return func(gmf service.GetMessageFunc) []string {
		for i, part := range text {
			matches := replyRegexp.FindAllStringSubmatch(part, -1)
			for _, match := range matches {
				variable := match[0]
				boardID := match[1]
				threadID, _ := strconv.Atoi(match[2])
				num, _ := strconv.Atoi(match[3])
				replace := ""
				pm, ok := gmf(&postMessageKey{boardID, threadID, num})
				if ok {
					replace = fmt.Sprintf(`<a href="%s">#%s%d</a>`, pm.Href(), strings.ToUpper(boardID), num)
				} else {
					replace = fmt.Sprintf(`#%s%d`, strings.ToUpper(boardID), num)
				}

				part = strings.Replace(part, variable, replace, -1)
			}

			text[i] = part
		}

		return text
	}
}

var tagRegexp = regexp.MustCompile(`<.*?>`)
var junkRegexp = regexp.MustCompile(`(?i)[^\wа-яё]`)

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
