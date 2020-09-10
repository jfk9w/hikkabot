package reddit

import "time"

type Media struct {
	RedditVideo struct {
		FallbackURL string `json:"fallback_url"`
	} `json:"reddit_video"`
}

type MediaContainer struct {
	Media       Media `json:"media"`
	SecureMedia Media `json:"secure_media"`
}

func (mc MediaContainer) FallbackURL() string {
	url := mc.Media.RedditVideo.FallbackURL
	if url == "" {
		url = mc.SecureMedia.RedditVideo.FallbackURL
	}

	return url
}

type ThingData struct {
	ID           uint64    `json:"-"`
	Created      time.Time `json:"-"`
	Title        string    `json:"title"`
	Subreddit    string    `json:"subreddit"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain"`
	URL          string    `json:"URL"`
	Ups          int       `json:"ups"`
	SelfTextHTML string    `json:"selftext_html"`
	IsSelf       bool      `json:"is_self"`
	CreatedSecs  float32   `json:"created_utc"`
	MediaContainer
	CrosspostParentList []MediaContainer `json:"crosspost_parent_list"`
	Permalink           string           `json:"permalink"`
}

func (d ThingData) PermalinkURL() string {
	return "https://reddit.com" + d.Permalink
}

type Thing struct {
	Data ThingData `json:"data"`
}
