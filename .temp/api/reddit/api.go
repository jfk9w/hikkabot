package reddit

import (
	"time"
)

type Thing struct {
	Data struct {
		Title         string  `json:"title"`
		Subreddit     string  `json:"subreddit"`
		Name          string  `json:"name"`
		Domain        string  `json:"domain"`
		URL           string  `json:"url"`
		RawCreatedUTC float32 `json:"created_utc"`
		Ups           int     `json:"ups"`

		CreatedUTC time.Time
		Extension  string
	} `json:"data"`
}

func (thing *Thing) init() {
	thing.Data.CreatedUTC = time.Unix(int64(thing.Data.RawCreatedUTC), 0)
}

type Sort = string

const (
	HotSort Sort = "hot"
	NewSort Sort = "new"
	TopSort Sort = "top"
)
