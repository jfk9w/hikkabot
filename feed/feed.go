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

func (id ID) SourceName(sources map[string]Source) string {
	if source, ok := sources[id.Source]; ok {
		return source.Name()
	} else {
		return id.Source
	}
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
	ID      ID
	Name    string
	RawData RawData
}

type Storage interface {
	Create(*Subscription) bool
	Get(ID) *Subscription
	Advance(telegram.ID) *Subscription
	Change(ID, Change) bool
	Active() []telegram.ID
	List(telegram.ID, bool) []Subscription
	Clear(telegram.ID, string) int
	Delete(ID) bool
}

type LogStorage interface {
	Log(id ID, attrs RawData) bool
}

func ToBytes(value interface{}) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

type Draft struct {
	ID   string
	Name string
}

var ErrDraftFailed = errors.New("Invalid syntax. Usage: `/sub` ENTITY [CHAT] [OPTIONS]")

type Source interface {
	ID() string
	Name() string
	Draft(command, options string, rawData RawData) (*Draft, error)
	Pull(pull *UpdatePull) error
}

type Update struct {
	RawData    []byte
	Text       format.Text
	Media      []*mediator.Future
	Attributes map[string]interface{}
}

type UpdatePull struct {
	Subscription
	queue  chan Update
	err    error
	cancel chan struct{}
}

func newUpdatePull(subscription Subscription) *UpdatePull {
	return &UpdatePull{
		Subscription: subscription,
		queue:        make(chan Update, 10),
		cancel:       make(chan struct{}),
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

func (p *UpdatePull) run(source Source) {
	defer close(p.queue)
	if err := source.Pull(p); err != nil {
		p.err = err
	}
}
