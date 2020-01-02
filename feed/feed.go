package feed

import (
	"encoding/json"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

type ID struct {
	ID     string
	ChatID telegram.ID
	Source string
}

func (id ID) String() string {
	return id.ChatID.String() + ":" + id.Source + ":" + id.ID
}

func ParseID(str string) (id ID, err error) {
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		err = errors.New("invalid ID")
	} else {
		id.ChatID, err = telegram.ParseID(parts[0])
		if err != nil {
			return
		}
		id.Source = parts[1]
		id.ID = parts[2]
	}
	return
}

type Subscription struct {
	ID     ID
	Name   string
	Item   []byte
	Offset int64
}

type Storage interface {
	Create(*Subscription) bool
	Get(ID) *Subscription
	Advance(telegram.ID) *Subscription
	Change(ID, Change) bool
	Active() []telegram.ID
}

func ToBytes(value interface{}) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func FromBytes(data []byte, value interface{}) {
	err := json.Unmarshal(data, value)
	if err != nil {
		panic(err)
	}
}

type Draft struct {
	ID   string
	Name string
	Item []byte
}

var ErrDraftFailed = errors.New("draft failed")

type Source interface {
	ID() string
	Draft(command, options string) (*Draft, error)
	Pull(pull *UpdatePull) error
}

type Update struct {
	Offset int64
	Text   format.Text
	Media  []*mediator.Future
}

type UpdatePull struct {
	Mediator *mediator.Mediator
	Offset   int64
	item     []byte
	queue    chan Update
	err      error
	cancel   chan struct{}
}

func newUpdatePull(item []byte, mediator *mediator.Mediator, offset int64) *UpdatePull {
	return &UpdatePull{
		Mediator: mediator,
		Offset:   offset,
		item:     item,
		queue:    make(chan Update, 10),
		cancel:   make(chan struct{}),
	}
}

func (p *UpdatePull) Submit(update Update) bool {
	select {
	case <-p.cancel:
		return false
	case p.queue <- update:
		return true
	}
}

func (p *UpdatePull) FromBytes(value interface{}) {
	FromBytes(p.item, value)
}

func (p *UpdatePull) run(source Source) {
	defer close(p.queue)
	if err := source.Pull(p); err != nil {
		p.err = err
	}
}
