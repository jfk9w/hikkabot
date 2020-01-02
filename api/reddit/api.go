package reddit

import (
	"time"
)

type Media struct {
	RedditVideo struct {
		FallbackURL string `json:"fallback_url"`
	} `json:"reddit_video"`
}

type MediaContainer struct {
	Media       Media `json:"media"`
	SecureMedia Media `json:"secure_media"`
}

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
		MediaContainer
		CrosspostParentList []MediaContainer `json:"crosspost_parent_list"`

		Created time.Time `json:"-"`
	} `json:"data"`
}

func (t *Thing) init() {
	t.Data.Created = time.Unix(int64(t.Data.CreatedUTC), 0)
}
