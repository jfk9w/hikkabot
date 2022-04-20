package feed

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"gopkg.in/guregu/null.v3"
)

type ID int64

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type Header struct {
	SubID  string `gorm:"primaryKey;column:sub_id"`
	Vendor string `gorm:"primaryKey"`
	FeedID ID     `gorm:"primaryKey"`
}

func (h Header) Labels() me3x.Labels {
	return make(me3x.Labels, 0, 3).
		Add("sub_id", h.SubID).
		Add("vendor", h.Vendor).
		Add("feed_id", h.FeedID)
}

func (h Header) String() string {
	return fmt.Sprintf("%d.%s.%s", h.FeedID, h.Vendor, h.SubID)
}

type Subscription struct {
	Header    `gorm:"embedded"`
	Name      string `gorm:"not null"`
	Data      gormf.JSONB
	UpdatedAt *time.Time
	Error     null.String
}

func (s *Subscription) TableName() string {
	return "feed"
}

type Draft struct {
	SubID string
	Name  string
	Data  any
}

type Event struct {
	Time   time.Time `gorm:"not null;index"`
	Type   string    `gorm:"not null;index:idx_event"`
	FeedID ID        `gorm:"not null;index:idx_event;column:chat_id"`
	Data   gormf.JSONB
}

func (e *Event) TableName() string {
	return "event"
}

type WriteHTML func(html *html.Writer) error

type Task func(context.Context) error

type MediaHash struct {
	FeedID     ID        `gorm:"primaryKey"`
	URL        string    `gorm:"not null"`
	Type       string    `gorm:"primaryKey;column:hash_type"`
	Value      string    `gorm:"primaryKey;column:hash"`
	FirstSeen  time.Time `gorm:"not null"`
	LastSeen   time.Time `gorm:"not null"`
	Collisions int64     `gorm:"not null"`
}

func (h *MediaHash) TableName() string {
	return "blob"
}
