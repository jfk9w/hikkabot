package dvach

import (
	"fmt"
	"strconv"
)

type File struct {
	DisplayName     string `json:"displayname"`
	Duration        string `json:"duration"`
	DurationSecs    int    `json:"duration_secs"`
	FullName        string `json:"fullname"`
	Height          int    `json:"height"`
	MD5             string `json:"md5"`
	Name            string `json:"name"`
	NSFW            int    `json:"nsfw"`
	Path            string `json:"path"`
	Size            int    `json:"size"`
	Thumbnail       string `json:"thumbnail"`
	ThumbnailHeight int    `json:"tn_height"`
	ThumbnailWidth  int    `json:"tn_width"`
	Type            int    `json:"type"`
	Width           int    `json:"width"`
}

func (f File) URL() string {
	return fmt.Sprintf("%s/%s", Endpoint, f.Path)
}

type Post struct {
	Banned    int    `json:"banned"`
	Closed    int    `json:"closed"`
	Comment   string `json:"comment"`
	Date      string `json:"date"`
	Email     string `json:"email"`
	Endless   int    `json:"endless"`
	Files     []File `json:"files"`
	LastHit   int    `json:"lasthit"`
	Name      string `json:"name"`
	Num       string `json:"num"`
	Op        int    `json:"op"`
	Parent    string `json:"parent"`
	Sticky    int    `json:"sticky"`
	Subject   string `json:"subject"`
	Tags      string `json:"tags"`
	Timestamp uint   `json:"timestamp"`
	Trip      string `json:"trip"`
	TripType  string `json:"trip_type"`
}

func (p Post) num() int {
	if n, err := strconv.Atoi(p.Num); err == nil {
		return n
	} else {
		panic(err)
	}
}
