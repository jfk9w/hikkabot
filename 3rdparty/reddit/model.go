package reddit

import (
	"html"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"
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

func (mc MediaContainer) FallbackURL() string {
	url := mc.Media.RedditVideo.FallbackURL
	if url == "" {
		url = mc.SecureMedia.RedditVideo.FallbackURL
	}

	return url
}

type ThingData struct {
	ID                  string      `json:"name" gorm:"primaryKey"`
	NumID               uint64      `json:"-" gorm:"not null"`
	CreatedAt           time.Time   `json:"-" gorm:"not null"`
	Title               string      `json:"title" gorm:"-"`
	Subreddit           string      `json:"subreddit" gorm:"not null;index"`
	Domain              string      `json:"domain" gorm:"not null"`
	URL                 null.String `json:"url"`
	Ups                 int         `json:"ups" gorm:"not null"`
	SelfTextHTML        string      `json:"selftext_html" gorm:"-"`
	IsSelf              bool        `json:"is_self" gorm:"not null"`
	CreatedSecs         float32     `json:"created_utc" gorm:"-"`
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

type Listing struct {
	Data struct {
		Children []Thing `json:"children"`
	} `json:"data"`
}

func (l *Listing) DecodeFrom(body io.Reader) error {
	if err := flu.DecodeFrom(flu.IO{R: body}, flu.JSON(l)); err != nil {
		return err
	}

	for i := range l.Data.Children {
		child := &l.Data.Children[i]
		var err error
		id := strings.Split(child.Data.ID, "_")[1]
		child.Data.NumID, err = strconv.ParseUint(id, 36, 64)
		if err != nil {
			return errors.Wrapf(err, "parse id: %s", id)
		}

		child.Data.SelfTextHTML = html.UnescapeString(child.Data.SelfTextHTML)
		child.Data.CreatedAt = time.Unix(int64(child.Data.CreatedSecs), 0)
	}

	return nil
}
