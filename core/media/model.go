package media

import (
	"context"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w-go/telegram-bot-api/ext/media"

	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"

	"github.com/jfk9w/hikkabot/core/feed"
)

type Storage interface {
	Alloc(now time.Time) (feed.Blob, error)
}

type Hash struct {
	FeedID     telegram.ID `gorm:"primaryKey"`
	URL        string      `gorm:"not null"`
	Type       string      `gorm:"primaryKey;column:hash_type"`
	Value      string      `gorm:"primaryKey;column:hash"`
	FirstSeen  time.Time   `gorm:"not null"`
	LastSeen   time.Time   `gorm:"not null"`
	Collisions int64       `gorm:"not null"`
}

func (h *Hash) TableName() string {
	return "blob"
}

type HashStorage interface {
	Check(ctx context.Context, hash *Hash) (bool, error)
}

type Metadata struct {
	Size     int64
	MIMEType string
}

func (m *Metadata) Handle(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type is empty")
	}

	var err error
	m.MIMEType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return errors.Wrapf(err, "invalid content type: %s", contentType)
	}

	contentLength := resp.Header.Get("Content-Length")
	m.Size, err = strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		m.Size = -1
	}

	return nil
}

type Resolver interface {
	GetClient(defaultClient *fluhttp.Client) *fluhttp.Client
	Resolve(ctx context.Context, client *fluhttp.Client, url string, maxSize int64) (string, error)
}

type Converter interface {
	Convert(ctx context.Context, ref *Ref) (media.Ref, error)
}

type Context struct {
	flu.Clock
	Storage
	metrics.Registry
	*Deduplicator
	HttpClient *fluhttp.Client
	SizeBounds [2]int64
	Converters map[string]Converter
	Retries    int
}