package feed

import (
	"context"
	"strings"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	null "gopkg.in/guregu/null.v3"
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
		"sub":    h.SubID,
		"vendor": h.Vendor,
		"feed":   h.FeedID,
	}
}

func (h *Header) String() string {
	return strings.Join([]string{h.FeedID.String(), h.Vendor, h.SubID}, HeaderSeparator)
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
	Data      gormutil.JSONB
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

type SubscriptionStorage interface {
	Active(ctx context.Context) ([]telegram.ID, error)
	Create(ctx context.Context, sub *Subscription) error
	Get(ctx context.Context, id *Header) (*Subscription, error)
	Shift(ctx context.Context, feedID telegram.ID) (*Subscription, error)
	List(ctx context.Context, feedID telegram.ID, active bool) ([]Subscription, error)
	DeleteAll(ctx context.Context, feedID telegram.ID, pattern string) (int64, error)
	Delete(ctx context.Context, header *Header) error
	Update(ctx context.Context, now time.Time, header *Header, value interface{}) error
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

type BlobHashStorage interface {
	Check(ctx context.Context, hash *BlobHash) error
}

type Storage interface {
	SubscriptionStorage
	BlobHashStorage
}

type Draft struct {
	SubID string
	Name  string
	Data  interface{}
}

type WriteHTML func(html *richtext.HTMLWriter) error

type Update struct {
	WriteHTML WriteHTML
	Data      gormutil.JSONB
	Error     error
}

type Loggable interface {
	Fields() logrus.Fields
}

type Queue struct {
	Header *Header
	C      chan Update
	data   gormutil.JSONB
}

func NewQueue(header *Header, data gormutil.JSONB, size int) *Queue {
	return &Queue{
		Header: header,
		C:      make(chan Update, size),
		data:   data,
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

func (q *Queue) Proceed(ctx context.Context, writeHTML WriteHTML, value interface{}) error {
	data, err := gormutil.ToJSONB(value)
	if err != nil {
		return err
	}

	update := Update{
		WriteHTML: writeHTML,
		Data:      data,
	}

	return q.submit(ctx, update)
}

func (q *Queue) Cancel(ctx context.Context, err error) error {
	return q.submit(ctx, Update{Error: err})
}

func (q *Queue) submit(ctx context.Context, update Update) error {
	select {
	case q.C <- update:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Vendor interface {
	Parse(ctx context.Context, ref string, options []string) (*Draft, error)
	Refresh(ctx context.Context, queue *Queue)
}
