package feed

import (
	"context"
	"strconv"
	"strings"
	"time"

	null "gopkg.in/guregu/null.v3"

	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrExists          = errors.New("exists")
	ErrForbidden       = errors.New("forbidden")
	ErrWrongVendor     = errors.New("wrong vendor")
	ErrSuspendedByUser = errors.New("suspended by user")
	ErrInvalidHeader   = errors.New("invalid header")
)

const HeaderSeparator = "+"

type Header struct {
	SubID  string      `gorm:"primaryKey;column:sub_id"`
	Vendor string      `gorm:"primaryKey"`
	FeedID telegram.ID `gorm:"primaryKey"`
}

func (h *Header) Fields() logrus.Fields {
	return logrus.Fields{
		"subID":  h.SubID,
		"vendor": h.Vendor,
		"feedID": h.FeedID,
	}
}

func (h *Header) String() string {
	return strings.Join([]string{strconv.FormatInt(int64(h.FeedID), 10), h.Vendor, h.SubID}, HeaderSeparator)
}

func (h *Header) MetricsLabels() metrics.Labels {
	return metrics.Labels{
		"sub_id", h.SubID,
		"vendor", h.Vendor,
		"feed_id", h.FeedID,
	}
}

func ParseHeader(value string) (*Header, error) {
	tokens := strings.Split(value, HeaderSeparator)
	if len(tokens) != 3 {
		return nil, ErrInvalidHeader
	}

	feedID, err := telegram.ParseID(tokens[0])
	if err != nil {
		return nil, errors.Wrapf(err, "invalid string id: %s", tokens[2])
	}

	return &Header{
		SubID:  tokens[2],
		Vendor: tokens[1],
		FeedID: feedID,
	}, nil
}

type Subscription struct {
	*Header   `gorm:"embedded"`
	Name      string `gorm:"not null"`
	Data      gormutil.Jsonb
	UpdatedAt *time.Time
	Error     null.String
}

func (s *Subscription) TableName() string {
	return "feed"
}

func (s *Subscription) Fields() logrus.Fields {
	fields := s.Header.Fields()
	fields["name"] = s.Name
	return fields
}

type BlobHash struct {
	FeedID     telegram.ID `gorm:"primaryKey;uniqueIndex:url_idx"`
	URL        string      `gorm:"not null;uniqueIndex:url_idx"`
	Type       string      `gorm:"primaryKey;column:hash_type"`
	Hash       string      `gorm:"primaryKey"`
	FirstSeen  time.Time   `gorm:"not null"`
	LastSeen   time.Time   `gorm:"not null"`
	Collisions int64       `gorm:"not null"`
}

func (h *BlobHash) TableName() string {
	return "blob"
}

type Storage interface {
	Init(ctx context.Context) ([]telegram.ID, error)
	Create(ctx context.Context, sub *Subscription) error
	Get(ctx context.Context, id *Header) (*Subscription, error)
	Shift(ctx context.Context, feedID telegram.ID) (*Subscription, error)
	List(ctx context.Context, feedID telegram.ID, active bool) ([]Subscription, error)
	DeleteAll(ctx context.Context, feedID telegram.ID, pattern string) (int64, error)
	Delete(ctx context.Context, header *Header) error
	Update(ctx context.Context, now time.Time, header *Header, value interface{}) error
	Check(ctx context.Context, hash *BlobHash) error
}

type Draft struct {
	SubID string
	Name  string
	Data  interface{}
}

type WriteHTML func(html *richtext.HTMLWriter) error

type Update struct {
	WriteHTML WriteHTML
	Data      gormutil.Jsonb
	Error     error
}

type Loggable interface {
	Fields() logrus.Fields
}

type Queue struct {
	Header  *Header
	data    gormutil.Jsonb
	channel chan Update
}

func NewQueue(header *Header, data gormutil.Jsonb, size int) *Queue {
	return &Queue{
		Header:  header,
		data:    data,
		channel: make(chan Update, size),
	}
}

func (q *Queue) GetData(ctx context.Context, value interface{}) error {
	if err := q.data.Unmarshal(value); err != nil {
		_ = q.Cancel(ctx, err)
		return err
	}

	return nil
}

func (q *Queue) Log(ctx context.Context, data Loggable) *logrus.Entry {
	return logrus.WithContext(ctx).
		WithFields(q.Header.Fields()).
		WithFields(data.Fields())
}

func (q *Queue) Proceed(ctx context.Context, writeHTML WriteHTML, data interface{}) error {
	jsonb, err := gormutil.ToJsonb(data)
	if err != nil {
		return err
	}

	update := Update{
		WriteHTML: writeHTML,
		Data:      jsonb,
	}

	return q.submit(ctx, update)
}

func (q *Queue) Cancel(ctx context.Context, err error) error {
	return q.submit(ctx, Update{Error: err})
}

func (q *Queue) submit(ctx context.Context, update Update) error {
	select {
	case q.channel <- update:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Vendor interface {
	Parse(ctx context.Context, ref string, options []string) (*Draft, error)
	Refresh(ctx context.Context, queue *Queue)
}
