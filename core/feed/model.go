package feed

import (
	"context"
	"strings"
	"time"

	gormutil "github.com/jfk9w-go/flu/gorm"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	null "gopkg.in/guregu/null.v3"

	telegram "github.com/jfk9w-go/telegram-bot-api"
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

func (h *Header) Labels() metrics.Labels {
	return metrics.Labels{}.
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
	Data      gormutil.JSONB
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
	Data      gormutil.JSONB
	Error     error
}

type HasLabels interface {
	Labels() metrics.Labels
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

func (q *Queue) Log(ctx context.Context, data HasLabels) *logrus.Entry {
	return logrus.WithContext(ctx).
		WithFields(q.Header.Labels().Map()).
		WithFields(data.Labels().Map())
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

type Blob interface {
	flu.Input
	flu.Output
}

type Event struct {
}
