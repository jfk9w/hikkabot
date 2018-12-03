package engine

import (
	"time"

	"github.com/jfk9w-go/hikkabot/content"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/schedx"
	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/logx"
	"github.com/jfk9w-go/red"
	"github.com/jfk9w-go/telegram"
)

type (
	Dvach     = *dvach.API
	Aconvert  = *aconvert.Balancer
	Red       = *red.API
	Scheduler = *schedx.T
	Telegram  = *telegram.T
)

var log = logx.Get("engine")

type Engine struct {
	Scheduler
	*DB
	ctx     *Context
	service feed.Service
}

func New(ctx *Context, interval time.Duration, dbFile string,
	redMetricsFile string, redMetricsChatID telegram.ChatID) *Engine {

	var engine = &Engine{
		Scheduler: schedx.New(interval),
		ctx:       ctx,
		DB:        OpenDB(dbFile).InitSchema(),
		service: &feed.GenericService{
			Typed: map[feed.Type]feed.Service{
				feed.DvachType: &feed.DvachService{
					Dvach:    ctx.Dvach,
					Aconvert: ctx.Aconvert,
				},
				feed.RedType: &feed.RedService{
					Red:           ctx.Red,
					MetricsFile:   redMetricsFile,
					MetricsChatID: redMetricsChatID,
				},
			},
		},
	}

	engine.Init(engine.Run)
	var active = engine.LoadActiveAccounts()
	log.Infof("Loading active accounts: %v", active)
	for _, chat := range active {
		engine.Schedule(chat)
	}

	return engine
}

func (engine *Engine) Start(id telegram.ChatID, state *feed.State) bool {
	if !engine.AppendState(id, state) {
		return false
	}

	log.Infof("Chat %v: start %v", id, state.ID)

	engine.Schedule(id)
	go engine.ctx.NotifyAdministrators(id, func(chat *telegram.Chat) string {
		return `#info
Subscription OK.
Chat: ` + content.FormatChatTitle(chat) + `
Thread: ` + engine.service.Title(state)
	})

	return true
}

func (engine *Engine) Suspend(id telegram.ChatID) bool {
	engine.Cancel(id)
	if !engine.DB.Suspend(id) {
		return false
	}

	log.Infof("Chat %v: suspend", id)

	go engine.ctx.NotifyAdministrators(id, func(chat *telegram.Chat) string {
		return `#info
All subscriptions suspended.
Chat: ` + content.FormatChatTitle(chat)
	})

	return true
}

func (engine *Engine) Run(id interface{}) {
	log.Debugf("Chat %v: loading state", id)

	var (
		chat  = id.(telegram.ChatID)
		state = engine.NextState(chat)
	)

	if state == nil {
		log.Debugf("Chat %v: empty", id)
		engine.Cancel(id)
		return
	}

	log.Debugf("Chat %v: state %v, offset %v", id, state.ID, state.Offset)

	var load, err = engine.service.Load(state)
	if !engine.CheckError(chat, state, err) {
		return
	}

	for load.HasNext() {
		var events = make(chan feed.Event)
		go load.Next(events)
		for event := range events {
			if err != nil {
				event.Interrupted()
				continue
			}

			switch event := event.(type) {
			case feed.Item:
				err = event.Send(engine.ctx.Telegram, chat)
				if err != nil {
					err = event.Retry(engine.ctx.Telegram, chat)
				}

				engine.CheckError(chat, state, err)

			case *feed.End:
				log.Debugf("Chat %v: persisting state %v, offset %v", id, state.ID, event.Offset)
				if !engine.PersistState(chat, state.WithOffset(event.Offset)) {
					log.Debugf("Chat %v: state %v interrupted", id, state.ID)
					return
				}
			}
		}

		if err != nil {
			return
		}
	}

	log.Debugf("Chat %v: no more events for state %v", id, state.ID)
	engine.PersistState(chat, state) // update for sort
}

func (engine *Engine) CheckError(chat telegram.ChatID, state *feed.State, err error) bool {
	if err == nil {
		return true
	}

	log.Debugf("Chat %v: persisting state %v with error (%v)", chat, state.ID, err)
	engine.PersistState(chat, state.WithError(err))
	go engine.ctx.NotifyAdministrators(chat, func(chat *telegram.Chat) string {
		return `#info
Subscription paused.
Chat: ` + content.FormatChatTitle(chat) + `
Title: ` + engine.service.Title(state) + `
Reason: ` + err.Error()
	})

	return false
}
