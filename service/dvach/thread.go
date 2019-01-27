package dvach

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/hikkabot/api/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/hikkabot/service"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"golang.org/x/exp/utf8string"
)

type threadOptions struct {
	BoardID        string `json:"board_id"`
	Num            int    `json:"num"`
	Title          string `json:"title"`
	UseNativeLinks bool   `json:"use_native_links"`
}

type PostKey struct {
	ChatID   telegram.ID
	BoardID  string
	ThreadID int
	Num      int
}

func (key *PostKey) String() string {
	return fmt.Sprintf("[ %s | %s | %d | %d ]", key.ChatID, key.BoardID, key.ThreadID, key.Num)
}

type MessageRef struct {
	Username  string
	MessageID telegram.ID
}

func (ref *MessageRef) Href() string {
	return fmt.Sprintf("https://t.me/%s/%d", ref.Username, ref.MessageID)
}

type PostMessageRefStorage interface {
	InsertPostRef(pk *PostKey, pm *MessageRef)
	GetPostRef(pk *PostKey) (*MessageRef, bool)
}

type ThreadService struct {
	agg      *service.Aggregator
	fs       service.FileSystemService
	storage  PostMessageRefStorage
	dvach    *dvach.Client
	aconvert *aconvert.Client
}

func Thread(
	agg *service.Aggregator, fs service.FileSystemService, storage PostMessageRefStorage,
	dvach *dvach.Client, aconvert *aconvert.Client) *ThreadService {
	return &ThreadService{agg, fs, storage, dvach, aconvert}
}

func (svc *ThreadService) ID() string {
	return "2ch/thread"
}

var threadRegexp = regexp.MustCompile(`^((http|https)://)?(2ch\.hk)?/([a-z]+)/res/([0-9]+)\.html?$`)

func (svc *ThreadService) Subscribe(input string, chat *service.EnrichedChat, args string) error {
	groups := threadRegexp.FindStringSubmatch(input)
	if len(groups) < 6 {
		return service.ErrInvalidFormat
	}

	boardID := groups[4]
	threadID, _ := strconv.Atoi(groups[5])

	post, err := svc.dvach.GetPost(boardID, threadID)
	if err != nil {
		return err
	}

	title := threadTitle(post)
	return svc.agg.Subscribe(chat, svc.ID(), post.BoardID+"/"+post.ParentString, title, &threadOptions{
		BoardID:        boardID,
		Num:            threadID,
		Title:          title,
		UseNativeLinks: chat.Type == telegram.Channel && chat.Username != nil,
	})
}

func (svc *ThreadService) Update(prevOffset int64, optionsFunc service.OptionsFunc, updatePipe *service.UpdatePipe) {
	defer updatePipe.Close()

	options := new(threadOptions)
	err := optionsFunc(options)
	if err != nil {
		updatePipe.Error(err)
		return
	}

	if prevOffset > 0 {
		prevOffset += 1
	}

	posts, err := svc.dvach.GetThread(options.BoardID, options.Num, int(prevOffset))
	if err != nil {
		updatePipe.Error(err)
		return
	}

	if len(posts) == 0 {
		return
	}

	for _, post := range posts {
		resources := make([]chan *flu.FileSystemResource, len(post.Files))
		for i, file := range post.Files {
			resources[i] = make(chan *flu.FileSystemResource)
			go svc.downloadFile(file, resources[i])
		}

		if !updatePipe.Submit(svc.updateBatchFunc(options, post, resources), int64(post.Num)) {
			return
		}
	}
}

var maxCaptionSize = telegram.MaxCaptionSize * 5 / 7

func (svc *ThreadService) updateBatchFunc(options *threadOptions, post *dvach.Post, resources []chan *flu.FileSystemResource) service.UpdateBatchFunc {
	parts := html.NewBuilder(maxHtmlChunkSize, -1).
		Parse(post.Comment).
		Build()

	return func(updateCh chan<- service.Update) {
		collapse := len(parts) == 1 && utf8string.NewString(parts[0]).RuneCount() <= maxCaptionSize && len(post.Files) == 1
		for i, part := range parts {
			i := i
			part := part
			uf := func(bot *telegram.Bot, chatID telegram.ID) (*telegram.Message, error) {
				matches := replyRegexp.FindAllStringSubmatch(part, -1)
				for _, match := range matches {
					variable := match[0]
					boardID := match[1]
					threadID, _ := strconv.Atoi(match[2])
					num, _ := strconv.Atoi(match[3])
					replace := ""
					if svc.storage != nil && options.UseNativeLinks {
						pm, ok := svc.storage.GetPostRef(&PostKey{chatID, boardID, threadID, num})
						if ok {
							replace = fmt.Sprintf(`<a href="%s">#%s%d</a>`, pm.Href(), strings.ToUpper(boardID), num)
						}
					}

					if replace == "" {
						replace = fmt.Sprintf(`#%s%d`, strings.ToUpper(boardID), num)
					}

					part = strings.Replace(part, variable, replace, -1)
				}

				update := new(service.GenericUpdate)
				if i == 0 {
					var maxSize int
					if collapse {
						maxSize = maxCaptionSize
					} else {
						maxSize = maxHtmlChunkSize
					}

					partsBuilder := html.NewBuilder(maxSize, 1).
						Text(`#` + options.Title).Br().
						Text(fmt.Sprintf(`#%s%d`, strings.ToUpper(post.BoardID), post.Num))
					if post.IsOriginal() {
						partsBuilder.Text(" #OP")
					}

					partsBuilder.Br()
					if collapse && len(post.Files) > 0 {
						file := post.Files[0]
						partsBuilder.Link(dvach.Host+file.Path, "[LINK]").Br()
						update.Entity = <-resources[i]
						update.Type = updateForFileType(post.Files[0].Type)
					}

					if part != "" {
						partsBuilder.Text("---").Br()
					}

					update.Text = partsBuilder.
						Parse(part).
						Build()[0]
				} else {
					update.Text = part
				}

				m, err := update.Send(bot, chatID)
				if err != nil {
					return nil, err
				}

				if i == 0 && m.Chat.Username != nil {
					svc.storage.InsertPostRef(
						&PostKey{chatID, post.BoardID, post.Parent, post.Num},
						&MessageRef{(*m.Chat.Username).String(), m.ID})
				}

				return m, nil
			}

			updateCh <- service.UpdateFunc(uf)
		}

		if collapse {
			return
		}

		for i, file := range post.Files {
			update := &service.GenericUpdate{
				Text: html.NewBuilder(telegram.MaxCaptionSize, 1).
					Link(dvach.Host+file.Path, "[LINK]").
					Build()[0],
			}

			resource := <-resources[i]
			if resource != nil {
				update.Entity = *resource
				update.Type = updateForFileType(file.Type)
			}

			updateCh <- update
		}
	}
}

func updateForFileType(fileType dvach.FileType) service.UpdateType {
	switch fileType {
	case dvach.WEBM, dvach.GIF, dvach.MP4:
		return service.VideoUpdate

	default:
		return service.PhotoUpdate
	}
}

var replyRegexp = regexp.MustCompile(`<a\s+href=".*?/([a-zA-Z0-9]+)/res/([0-9]+)\.html#([0-9]+)".*?>.*?</a>`)

func (svc *ThreadService) downloadFile(file *dvach.File, ch chan<- *flu.FileSystemResource) {
	defer close(ch)

	resource := svc.fs.NewTempResource()
	err := svc.dvach.DownloadFile(file, resource)
	if err != nil {
		_ = os.RemoveAll(resource.Path())
		log.Printf("Failed to download file %s: %s", file.Path, err)
		ch <- nil
		return
	}

	if file.Type == dvach.WEBM {
		start := time.Now()
		aresp, err := svc.aconvert.Convert(resource, aconvert.NewOpts().
			TargetFormat("mp4").
			VideoOptionSize(0).
			Code(81000))

		duration := time.Now().Sub(start)
		if stat, err := os.Stat(resource.Path()); err == nil {
			// just in case
			log.Println("Conversion took", duration.Seconds(), "secs, file size", stat.Size()>>10, "KB")
		}

		_ = os.RemoveAll(resource.Path())

		if err != nil {
			log.Printf("Failed to convert file %s: %s", file.Path, err)
			ch <- nil
			return
		}

		err = svc.aconvert.Download(aresp, resource)
		if err != nil {
			_ = os.RemoveAll(resource.Path())
			log.Printf("Failed to download %+v: %s", aresp, err)
			ch <- nil
			return
		}
	}

	sizeLimit := service.MaxPhotoSize
	switch file.Type {
	case dvach.WEBM, dvach.GIF, dvach.MP4:
		sizeLimit = service.MaxVideoSize
	}

	stat, err := os.Stat(resource.Path())
	if err != nil {
		_ = os.RemoveAll(resource.Path())
		log.Printf("Failed to stat file %s: %s", resource.Path(), err)
		ch <- nil
		return
	}

	if stat.Size() > int64(sizeLimit) {
		_ = os.RemoveAll(resource.Path())
		log.Printf("File %s exceeds size limit %d", resource.Path(), sizeLimit)
		ch <- nil
		return
	}

	ch <- &resource
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
