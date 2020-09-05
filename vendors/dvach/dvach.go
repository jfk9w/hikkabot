package dvach

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/vendors/common"
)

const (
	Domain = "2ch.hk"
	Host   = "https://" + Domain
)

type FileType int

const (
	JPEG FileType = 1
	PNG  FileType = 2
	GIF  FileType = 4
	WebM FileType = 6
	MP4  FileType = 10
)

var Type2MIMEType = map[FileType]string{
	JPEG: "image/jpeg",
	PNG:  "image/png",
	GIF:  "image/gif",
	WebM: "video/webm",
	MP4:  "video/mp4",
}

func (ft FileType) MIMEType() string {
	return Type2MIMEType[ft]
}

type File struct {
	Path         string   `json:"path"`
	Type         FileType `json:"type"`
	Size         int      `json:"size"`
	DurationSecs *int     `json:"duration_secs"`
	Width        *int     `json:"width"`
	Height       *int     `json:"height"`
}

func (f File) URL() string {
	return Host + f.Path
}

func (f File) WriteMedia(html *format.HTMLWriter, collapsible bool) {
	html.Media(f.URL(), common.NewResolvedMediaRef(
		f.Type.MIMEType(), flu.URL(f.URL())), collapsible)
}

type Post struct {
	NumString    string `json:"num"`
	ParentString string `json:"parent"`
	DateString   string `json:"date"`
	Subject      string `json:"subject"`
	Comment      string `json:"comment"`
	Files        []File `json:"files"`

	// OP-only fields
	PostsCount *int `json:"posts_count"`
	FilesCount *int `json:"files_count"`

	// fields with custom initialization
	Board  string
	Num    int
	Parent int
	Date   time.Time
}

func (p Post) WriteMedia(html *format.HTMLWriter, text string, collapsible bool) {
	for _, file := range p.Files {
		html.Text(text).Media(file.URL(), common.NewResolvedMediaRef(
			file.Type.MIMEType(), flu.URL(file.URL())), collapsible)
	}
}

var (
	tz     *time.Location
	tzOnce sync.Once
)

func (p *Post) init(board string) (err error) {
	tzOnce.Do(func() {
		loc, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			panic(err)
		}
		tz = loc
	})
	p.Board = board
	p.Num, err = strconv.Atoi(p.NumString)
	if err != nil {
		return
	}
	p.Parent, err = strconv.Atoi(p.ParentString)
	if err != nil {
		return
	}
	if p.Parent == 0 {
		p.Parent = p.Num
		p.ParentString = p.NumString
	}
	datestr := []rune(p.DateString)
	p.Date, err = time.ParseInLocation("02/01/06 15:04:05",
		string(datestr[:8])+string(datestr[12:]), tz)
	return err
}

func (p *Post) IsOriginal() bool {
	return p.Parent == p.Num
}

func (p *Post) URL() string {
	if p.IsOriginal() {
		return fmt.Sprintf("%s/%s/res/%s.html", Host, p.Board, p.NumString)
	}
	return fmt.Sprintf("%s/%s/res/%s.html#%s", Host, p.Board, p.ParentString, p.NumString)
}

type Posts []Post

func (ps Posts) init(board string) (err error) {
	for i := range ps {
		err = (&ps[i]).init(board)
		if err != nil {
			return
		}
	}
	return
}

type Catalog struct {
	BoardName string `json:"BoardName"`
	Threads   []Post `json:"threads"`
}

func (c *Catalog) init(boardID string) (err error) {
	return Posts(c.Threads).init(boardID)
}

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Error struct {
	Code int    `json:"Code"`
	Err  string `json:"Error"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Err)
}

func cookies(usercode string, path string) []*http.Cookie {
	return []*http.Cookie{
		{
			Name:   "usercode_auth",
			Value:  usercode,
			Domain: Domain,
			Path:   path,
		},
		{
			Name:   "ageallow",
			Value:  "1",
			Domain: Domain,
			Path:   path,
		},
	}
}
