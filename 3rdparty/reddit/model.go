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
	ID                  uint64    `json:"-" gorm:"primaryKey;not null;autoIncrement:false"`
	CreatedAt           time.Time `json:"-" gorm:"not null"`
	Title               string    `json:"title" gorm:"-"`
	Subreddit           string    `json:"subreddit" gorm:"not null"`
	Name                string    `json:"name" gorm:"-"`
	Domain              string    `json:"domain" gorm:"not null"`
	URL                 string    `json:"URL" gorm:"-"`
	Ups                 int       `json:"ups" gorm:"not null"`
	SelfTextHTML        string    `json:"selftext_html" gorm:"-"`
	IsSelf              bool      `json:"is_self" gorm:"-"`
	CreatedSecs         float32   `json:"created_utc" gorm:"-"`
	MediaContainer      `gorm:"-"`
	CrosspostParentList []MediaContainer `json:"crosspost_parent_list" gorm:"-"`
	Permalink           string           `json:"permalink" gorm:"-"`
	Author              string           `json:"author" gorm:"not null"`
}

func (d ThingData) PermalinkURL() string {
	return "https://reddit.com" + d.Permalink
}

type Thing struct {
	Data     ThingData `json:"data" gorm:"embedded"`
	LastSeen time.Time `json:"-" gorm:"not null"`
}

func (t Thing) TableName() string {
	return "reddit"
}
