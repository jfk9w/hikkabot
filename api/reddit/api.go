package reddit

import (
	"time"
)

type Thing struct {
	Data struct {
		Title        string  `json:"title"`
		Subreddit    string  `json:"subreddit"`
		Name         string  `json:"name"`
		Domain       string  `json:"domain"`
		URL          string  `json:"URL"`
		CreatedUTC   float32 `json:"created_utc"`
		Ups          int     `json:"ups"`
		SelfTextHTML string  `json:"selftext_html"`
		IsSelf       bool    `json:"is_self"`

		Created     time.Time `json:"-"`
		ResolvedURL string    `json:"-"`
		Extension   string    `json:"-"`
	} `json:"data"`
}

func (t *Thing) init() {
	t.Data.Created = time.Unix(int64(t.Data.CreatedUTC), 0)
}

type Sort = string

const (
	Hot Sort = "hot"
	New Sort = "new"
	Top Sort = "top"
)
