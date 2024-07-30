package dvach

import (
	"fmt"
	"sync"
	"time"
)

type File struct {
	Path         string   `json:"path"`
	Type         FileType `json:"type"`
	Size         int64    `json:"size"`
	DurationSecs *int     `json:"duration_secs"`
	Width        *int     `json:"width"`
	Height       *int     `json:"height"`
}

func (f File) URL() string {
	return Host + f.Path
}

type Post struct {
	Num        int    `json:"num"`
	Parent     int    `json:"parent"`
	DateString string `json:"date"`
	Subject    string `json:"subject"`
	Comment    string `json:"comment"`
	Files      []File `json:"files"`

	// OP-only fields
	PostsCount *int `json:"posts_count"`
	FilesCount *int `json:"files_count"`

	// fields with custom initialization
	Board string
	Date  time.Time
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
	if p.Parent == 0 {
		p.Parent = p.Num
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
		return fmt.Sprintf("%s/%s/res/%d.html", Host, p.Board, p.Num)
	}
	return fmt.Sprintf("%s/%s/res/%d.html#%d", Host, p.Board, p.Parent, p.Num)
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
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}
