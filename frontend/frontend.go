package frontend

import (
	"strings"
	"sync"

	"io"

	"time"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/backend"
	"github.com/jfk9w-go/hikkabot/bot"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
)

type (
	Bot interface {
		io.Closer
		telegram.Updater
		telegram.Api
		bot.Frontend
	}
)

func Run(token string) io.Closer {
	f := &frontend{
		waitGroup: &sync.WaitGroup{},
	}

	b := bot.NewAugmentedBot(token, f)
	d := dvach.New(httpx.DefaultClient)
	w := aconvert.WithCache(3*24*time.Hour, 30*time.Second, 12*time.Hour)

	f.bot = b
	f.back = backend.New(b, d, w)
	go f.run()

	return misc.BroadcastCloser(b, w)
}

var log = logrus.GetLogger("frontend")

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
