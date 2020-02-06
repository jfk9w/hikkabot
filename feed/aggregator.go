package feed

import (
	"expvar"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/mediator/request"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

type Aggregator struct {
	Channel
	Storage
	*mediator.Mediator
	Timeout time.Duration
	Aliases map[telegram.Username]telegram.ID
	AdminID telegram.ID
	sources map[string]Source
	chats   map[telegram.ID]bool
	mu      sync.RWMutex
	metrics *expvar.Map
}

func (a *Aggregator) AddSource(source Source) *Aggregator {
	if a.sources == nil {
		a.sources = make(map[string]Source)
	}
	a.sources[source.ID()] = source
	return a
}

func (a *Aggregator) Init() *Aggregator {
	a.metrics = expvar.NewMap("aggregator")
	a.chats = make(map[telegram.ID]bool)
	for _, chatID := range a.Active() {
		a.RunFeed(chatID)
	}
	return a
}

func (a *Aggregator) runUpdater(chatID telegram.ID) {
	sub := a.Advance(chatID)
	if sub == nil {
		// no next item - subscriptions exhausted, stopping the updater
		a.mu.Lock()
		delete(a.chats, chatID)
		a.mu.Unlock()
		log.Printf("Stopped updater for %v", chatID)
		a.metrics.Add("active", -1)
		return
	}

	err := a.pullUpdates(chatID, *sub)
	if err != nil {
		if err == ErrNotFound {
			// the storage update has failed
			// meaning the subscription was suspended by external source
			// we don't need to do anything else
		} else {
			a.change(0, sub.ID, Change{Error: err})
		}
	}

	// reschedule the updater
	time.AfterFunc(a.Timeout, func() { a.runUpdater(chatID) })
}

func (a *Aggregator) pullUpdates(chatID telegram.ID, sub Subscription) error {
	source, ok := a.sources[sub.ID.Source]
	if !ok {
		return errors.Errorf("no such source: %s", sub.ID.Source)
	}
	pull := newUpdatePull(sub)
	go pull.run(source)
	hasUpdates := false
	for update := range pull.queue {
		hasUpdates = true
		err := a.SendUpdate(chatID, update)
		if err != nil {
			return errors.Wrapf(err, "send update: %+v", update)
		}
		a.metrics.Add("updates", 1)
		err = a.change(0, sub.ID, Change{RawData: update.RawData})
		if err != nil {
			pull.cancel <- struct{}{}
			close(pull.cancel)
			return err
		}
	}
	if pull.err != nil {
		return errors.Wrap(pull.err, "pull updates")
	}
	if !hasUpdates {
		if err := a.change(0, sub.ID, Change{RawData: sub.RawData.Bytes()}); err != nil {
			return err
		}
	}
	return nil
}

func (a *Aggregator) RunFeed(chatID telegram.ID) {
	// check that the feed does not exist yet
	// via double-checked locking
	a.mu.RLock()
	ok := a.chats[chatID]
	a.mu.RUnlock()
	if ok {
		return
	}
	a.mu.Lock()
	if a.chats[chatID] {
		a.mu.Unlock()
		return
	}
	a.chats[chatID] = true
	a.mu.Unlock()
	// run updater
	go a.runUpdater(chatID)
	a.metrics.Add("active", 1)
	log.Printf("Started updater for %v", chatID)
}

var ErrNotFound = errors.New("not found")

func (a *Aggregator) change(userID telegram.ID, id ID, change Change) error {
	ctx := &changeContext{chatID: id.ChatID}
	if err := ctx.checkAccess(a, userID); err != nil {
		return err
	}
	ok := a.Storage.Change(id, change)
	if !ok {
		return ErrNotFound
	}
	if change.RawData != nil {
		// if this is an offset update we don't need to notify admins
		log.Printf("Updated raw data for %s to [...]", id)
		return nil
	} else {
		if change.Error == nil {
			log.Printf("Resumed %s", id)
		} else {
			log.Printf("Suspended %s (reason: '%s')", id, change.Error)
		}
	}

	if change.Error == nil {
		a.RunFeed(id.ChatID)
	}

	sub := a.Get(id)
	if sub == nil {
		return ErrNotFound
	}

	// notifications
	adminIDs, err := ctx.getAdminIDs(a)
	if err != nil {
		return err
	}
	status := "OK 🔥"
	if change.Error != nil {
		status = "suspended"
	}

	title, _ := ctx.getChatTitle(a)
	text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil).
		Text("Subscription " + status).NewLine().
		Text("Chat: " + title).NewLine().
		Text("Service: " + id.SourceName(a.sources)).NewLine().
		Text("Item: " + sub.Name)
	var button telegram.ReplyMarkup
	if change.Error != nil {
		button = telegram.InlineKeyboard("Resume", "r", id.String())
		text.NewLine().
			Text("Reason: " + change.Error.Error())
	} else {
		button = telegram.InlineKeyboard(
			"Suspend", "s", id.String(),
			"Delete", "d", id.String())
	}

	go a.SendAlert(adminIDs, text.Format(), button)
	return nil
}

func (a *Aggregator) changeByUser(tg telegram.Client, c *telegram.Command, change Change) error {
	reply := "OK"
	id, err := ParseID(c.Payload)
	if err == nil {
		if err = a.change(c.User.ID, id, change); err != nil {
			reply = err.Error()
		}
	} else {
		reply = "failed to parse ID"
	}
	return c.Reply(tg, reply)
}

func (a *Aggregator) createChangeContext(c *telegram.Command, fields []string, chatIdx int) (*changeContext, error) {
	ctx := new(changeContext)
	if len(fields) > chatIdx && fields[chatIdx] != "." {
		username := telegram.Username(fields[chatIdx])
		var chatID telegram.ChatID = username
		if unaliased, ok := a.Aliases[username]; ok {
			chatID = unaliased
		}
		ctx.chatID = chatID
	} else {
		ctx.chatID = c.Chat.ID
		ctx.chat = c.Chat
	}
	return ctx, ctx.checkAccess(a, c.User.ID)
}

func (a *Aggregator) doCreate(c *telegram.Command) error {
	fields := strings.Fields(c.Payload)
	cmd := fields[0]
	ctx, err := a.createChangeContext(c, fields, 1)
	if err != nil {
		return err
	}
	options := ""
	if len(fields) > 2 {
		options = fields[2]
	}
	rawData := NewRawData(nil)
	for sourceID, source := range a.sources {
		var draft *Draft
		draft, err = source.Draft(cmd, options, rawData)
		switch err {
		case ErrDraftFailed:
			continue
		case nil:
			id := ID{
				ID:     draft.ID,
				ChatID: ctx.chat.ID,
				Source: sourceID,
			}
			if len(id.String()) > 62 {
				return errors.New("ID too long")
			}
			ctx := &Subscription{
				ID:      id,
				Name:    draft.Name,
				RawData: rawData,
			}
			if a.Storage.Create(ctx) {
				return a.change(0, ctx.ID, Change{})
			} else {
				return errors.New("exists")
			}
		default:
			return err
		}
	}
	return err
}

func (a *Aggregator) Create(tg telegram.Client, c *telegram.Command) error {
	if err := a.doCreate(c); err != nil {
		return c.Reply(tg, err.Error())
	} else {
		return nil
	}
}

func (a *Aggregator) Resume(tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(tg, c, Change{})
}

var ErrSuspendedByUser = errors.New("suspended by user")

func (a *Aggregator) Suspend(tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(tg, c, Change{Error: ErrSuspendedByUser})
}

func (a *Aggregator) Delete(tg telegram.Client, c *telegram.Command) error {
	id, err := ParseID(c.Payload)
	if err != nil {
		return c.Reply(tg, "failed to parse ID")
	}
	ctx := &changeContext{chatID: id.ChatID}
	if err := ctx.checkAccess(a, c.User.ID); err != nil {
		return c.Reply(tg, err.Error())
	}
	if a.Storage.Delete(id) {
		return c.Reply(tg, "OK")
	} else {
		return c.Reply(tg, "not found")
	}
}

func (a *Aggregator) Status(tg telegram.Client, c *telegram.Command) error {
	text := format.NewHTML(telegram.MaxMessageSize, 0, nil, nil)
	if c.User.ID == a.AdminID {
		expvar.Do(func(kv expvar.KeyValue) {
			if kv.Key == "cmdline" || kv.Key == "memstats" {
				return
			}
			text.NewLine().Text(kv.Key + ": " + kv.Value.String())
		})
	} else {
		text.Text("OK")
	}
	a.SendAlert([]telegram.ID{c.Chat.ID}, text.Format(), nil)
	return nil
}

func (a *Aggregator) YouTube(tg telegram.Client, c *telegram.Command) error {
	req := &request.Youtube{URL: c.Payload, MaxSize: mediator.MaxSize(telegram.Video)[1]}
	err := a.SendUpdate(c.Chat.ID, Update{
		Text: format.Text{
			Pages:     []string{""},
			ParseMode: telegram.HTML,
		},
		Media: []*mediator.Future{a.SubmitMedia(c.Payload, req)},
	})
	if err != nil {
		err = c.Reply(tg, err.Error())
	}
	return err
}

func (a *Aggregator) List(tg telegram.Client, c *telegram.Command) error {
	fields := strings.Fields(c.Payload)
	ctx, err := a.createChangeContext(c, fields, 0)
	if err != nil {
		return err
	}
	active := false
	command := "resume"
	if len(fields) > 1 && fields[1] != "r" {
		active = true
		command = "suspend"
	}
	subs := a.Storage.List(ctx.chat.ID, active)
	if (len(fields) < 2 || fields[1] == "") && len(subs) == 0 {
		active = true
		command = "suspend"
		subs = a.Storage.List(ctx.chat.ID, active)
	}
	keyboard := make([]string, len(subs)*3)
	for i, sub := range subs {
		keyboard[3*i] = "[" + sub.ID.SourceName(a.sources) + "] " + sub.Name
		keyboard[3*i+1] = command[:1]
		keyboard[3*i+2] = sub.ID.String()
	}
	title, _ := ctx.getChatTitle(a)
	a.SendAlert(
		[]telegram.ID{c.Chat.ID},
		format.NewHTML(0, 0, nil, nil).
			Text("Chat: ").Text(title).
			NewLine().
			Text(strconv.Itoa(len(subs))).
			Text(" subscriptions eligible for ").
			Tag("b").Text(command).EndTag().
			Format(),
		telegram.InlineKeyboard(keyboard...))
	return nil
}

func (a *Aggregator) Clear(tg telegram.Client, c *telegram.Command) error {
	space := strings.Index(c.Payload, " ")
	if space < 0 || len(c.Payload) == space+1 {
		return c.Reply(tg, "this command requires two arguments")
	}
	fields := [2]string{c.Payload[:space], c.Payload[space+1:]}
	ctx, err := a.createChangeContext(c, fields[:], 0)
	if err != nil {
		return err
	}
	cleared := a.Storage.Clear(ctx.chat.ID, fields[1])
	title, _ := ctx.getChatTitle(a)
	a.SendAlert(
		[]telegram.ID{c.Chat.ID},
		format.NewHTML(0, 0, nil, nil).
			Text("Chat: ").Text(title).
			NewLine().
			Text(strconv.Itoa(cleared)).
			Text(" subscriptions ").
			Tag("b").Text("cleared").EndTag().
			Format(),
		nil)
	return nil
}

func (a *Aggregator) CommandListener(username string) *telegram.CommandListener {
	return telegram.NewCommandListener(username).
		HandleFunc("/sub", a.Create).
		HandleFunc("r", a.Resume).
		HandleFunc("s", a.Suspend).
		HandleFunc("d", a.Delete).
		HandleFunc("/status", a.Status).
		HandleFunc("/youtube", a.YouTube).
		HandleFunc("/list", a.List).
		HandleFunc("/clear", a.Clear)
}
