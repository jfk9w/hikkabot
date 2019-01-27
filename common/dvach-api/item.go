package dvach

import (
	"fmt"
	"strings"
	"time"
)

type (
	Board = string
	Num   = int
)

type Ref struct {
	NumString string `json:"num"`

	// init
	Board Board
	Num   Num
}

func (ref *Ref) init(board string) bool {
	var num, ok = ToNum(ref.NumString)
	if !ok {
		return false
	}

	ref.Board = strings.ToLower(board)
	ref.Num = num
	return true
}

func (ref Ref) String() string {
	return fmt.Sprintf("%s%s", ref.Board, ref.NumString)
}

type File struct {
	Path         string `json:"path"`
	Type         int    `json:"type"`
	Size         int    `json:"size"`
	DurationSecs *int   `json:"duration_secs"`
	Width        *int   `json:"width"`
	Height       *int   `json:"height"`
}

func (f *File) URL() string {
	return Endpoint + f.Path
}

type Item struct {
	Ref
	Subject    string  `json:"subject"`
	DateString string  `json:"date"`
	Comment    string  `json:"comment"`
	Files      []*File `json:"files"`

	// init
	Date time.Time
}

func (item *Item) init(board string) bool {
	var date, ok = ToTime(item.DateString)
	if !ok {
		return false
	}

	if !item.Ref.init(board) {
		return false
	}

	item.Date = date
	return true
}

type Thread struct {
	Item
	PostsCount int `json:"posts_count"`
	FilesCount int `json:"files_count"`
}

type Post struct {
	Item
	ParentString string `json:"parent"`

	// init
	Parent Num
}

func (p *Post) init(board string) bool {
	var parentString = p.ParentString
	if parentString == "0" {
		parentString = p.NumString
	}

	var parent, ok = ToNum(p.ParentString)
	if !ok {
		return false
	}

	if !p.Item.init(board) {
		return false
	}

	p.Parent = parent
	return true
}

func (p *Post) ParentRef() Ref {
	if p.Parent != 0 {
		return Ref{p.ParentString, p.Board, p.Parent}
	}

	return p.Ref
}

type Catalog struct {
	Threads []*Thread `json:"threads"`
}

func (c *Catalog) init(board Board) {
	for _, thread := range c.Threads {
		thread.init(board)
	}
}
