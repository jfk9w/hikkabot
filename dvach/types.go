package dvach

import (
	"fmt"
	"strconv"
)

type File struct {
	DisplayName string `json:"displayname"`
	Path        string `json:"path"`
	Type        int    `json:"type"`
}

func (f File) URL() string {
	return fmt.Sprintf("%s%s", Endpoint, f.Path)
}

type Post struct {
	Closed     int    `json:"closed"`
	Comment    string `json:"comment"`
	Endless    int    `json:"endless"`
	Files      []File `json:"files"`
	Num        string `json:"num"`
	PostsCount int    `json:"posts_count"`
	Sticky     int    `json:"sticky"`
	Subject    string `json:"subject"`
}

func (p Post) NumInt() int {
	if n, err := strconv.Atoi(p.Num); err == nil {
		return n
	} else {
		panic(err)
	}
}

type Catalog struct {
	Threads []Post `json:"threads"`
}
