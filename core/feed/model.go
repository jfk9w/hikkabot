package feed

import (
	"context"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
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

func (h *Header) Labels() me3x.Labels {
	return me3x.Labels{}.
		Add("feed", h.FeedID).
		Add("vendor", h.Vendor).
		Add("sub", h.SubID)
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
	Data      gormf.JSONB
	UpdatedAt *time.Time
	Error     null.String
}

func (s *Subscription) TableName() string {
	return "feed"
}

type Storage interface {
	Active(ctx context.Context) ([]telegram.ID, error)
	Create(ctx context.Context, sub *Subscription) error
	Get(ctx context.Context, id *Header) (*Subscription, error)
	Shift(ctx context.Context, feedID telegram.ID) (*Subscription, error)
	List(ctx context.Context, feedID telegram.ID, active bool) ([]Subscription, error)
	DeleteAll(ctx context.Context, feedID telegram.ID, pattern string) (int64, error)
	Delete(ctx context.Context, header *Header) error
	Update(ctx context.Context, now time.Time, header *Header, value interface{}) error
}

type Draft struct {
	SubID string
	Name  string
	Data  interface{}
}

type WriteHTML func(html *html.Writer) error

type Update struct {
	WriteHTML WriteHTML
	Data      gormf.JSONB
	Error     error
}

type Queue struct {
	Header *Header
	C      chan Update
	data   gormf.JSONB
}

func NewQueue(header *Header, data gormf.JSONB, size int) *Queue {
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

func (q *Queue) Log(ctx context.Context, data me3x.Labeled) *logrus.Entry {
	return logrus.WithContext(ctx).
		WithFields(q.Header.Labels().Map()).
		WithFields(data.Labels().Map())
}

func (q *Queue) Proceed(ctx context.Context, writeHTML WriteHTML, value interface{}) error {
	data, err := gormf.ToJSONB(value)
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

type Blob interface {
	flu.Input
	flu.Output
}
