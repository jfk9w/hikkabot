package frontend

import (
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Backend interface {
		Subscribe(telegram.ChatRef, dvach.ID, string, int) error
		Unsubscribe(telegram.ChatRef, dvach.ID) error
		UnsubscribeAll(telegram.ChatRef) error
	}

	Bot interface {
		GetMe() (*telegram.User, error)
		UpdateChannel() <-chan telegram.Update
		SendText(telegram.ChatRef, string, ...interface{})
		GetAdmins(telegram.ChatRef) ([]telegram.ChatRef, error)
		NotifyAll([]telegram.ChatRef, string, ...interface{})
	}

	Dvach interface {
		Post(dvach.ID) (*dvach.Post, error)
	}
)

func Run(bot Bot, dvch Dvach, back Backend) {
	front := &T{bot, dvch, back}
	go front.run()
}

var log = logrus.GetLogger("T")

type ParsedCommand struct {
	Command string
	Params  []string
}

func (ps ParsedCommand) String() string {
	sb := &strings.Builder{}
	sb.WriteRune('/')
	sb.WriteString(ps.Command)
	for _, param := range ps.Params {
		sb.WriteRune(' ')
		sb.WriteString(param)
	}

	return sb.String()
}
