package feed

import (
	"expvar"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/media"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/pkg/errors"
)

type Aggregator struct {
	Channel
	Subscription
	Storage
	Timeout time.Duration
	Media   *media.Manager
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

	err := a.pullUpdates(chatID, sub)
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

func (a *Aggregator) pullUpdates(chatID telegram.ID, sub *Subscription) error {
	source, ok := a.sources[sub.ID.Source]
	if !ok {
		return errors.Errorf("no such source: %s", sub.ID.Source)
	}
	pull := newUpdatePull(sub.Item, a.Media, sub.Offset)
	go pull.run(source)
	hasUpdates := false
	for update := range pull.queue {
		hasUpdates = true
		err := a.SendUpdate(chatID, update)
		if err != nil {
			return errors.Wrapf(err, "send update: %+v", update)
		}
		a.metrics.Add("updates", 1)
		err = a.change(0, sub.ID, Change{Offset: update.Offset})
		if err != nil {
			pull.cancel <- struct{}{}
			close(pull.cancel)
			return err
		}
		sub.Offset = update.Offset
	}
	if pull.err != nil {
		return errors.Wrap(pull.err, "pull updates")
	}
	if !hasUpdates {
		if err := a.change(0, sub.ID, Change{Offset: sub.Offset}); err != nil {
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
	var ctx *changeContext
	if userID != 0 {
		// if the change is generated by a real user, check its privelege now and create the context
		ctx = &changeContext{chatID: id.ChatID}
		if err := ctx.checkAccess(a, userID); err != nil {
			return err
		}
	}
	ok := a.Storage.Change(id, change)
	if !ok {
		return ErrNotFound
	}
	if change.Offset != 0 {
		// if this is an offset update we don't need to notify admins
		log.Printf("Updated offset for %s to %d", id, change.Offset)
		return nil
	}
	if ctx == nil {
		// if the change is generated by a system user, we need to create the context now
		ctx = &changeContext{chatID: id.ChatID}
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
		status = "suspend"
	}

	title := "<private>"
	chat, _ := ctx.getChat(a)
	if chat.Type != telegram.PrivateChat {
		title = chat.Title
	}

	text := format.NewHTML(telegram.MaxMessageSize, 1, nil, nil).
		Text("Subscription " + status).NewLine().
		Text("Chat: " + title).NewLine().
		Text("Service: " + id.Source).NewLine().
		Text("Item: " + sub.Name)
	var button telegram.ReplyMarkup
	if change.Error != nil {
		button = telegram.CommandButton("Resume", "resume", id.String())
		text.NewLine().
			Text("Reason: " + change.Error.Error())
	} else {
		button = telegram.CommandButton("Suspend", "suspend", id.String())
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
	_, err = tg.AnswerCallbackQuery(c.CallbackQueryID,
		&telegram.AnswerCallbackQueryOptions{Text: reply})
	return err
}

func (a *Aggregator) doCreate(c *telegram.Command) error {
	fields := strings.Fields(c.Payload)
	cmd := fields[0]
	ctx := new(changeContext)
	if len(fields) > 1 && fields[1] != "." {
		username := telegram.Username(fields[1])
		var chatID telegram.ChatID = username
		if unaliased, ok := a.Aliases[username]; ok {
			chatID = unaliased
		}
		ctx.chatID = chatID
	} else {
		ctx.chatID = c.Chat.ID
		ctx.chat = c.Chat
	}
	if err := ctx.checkAccess(a, c.User.ID); err != nil {
		return err
	}
	options := ""
	if len(fields) > 2 {
		options = fields[2]
	}
	for sourceID, source := range a.sources {
		draft, err := source.Draft(cmd, options)
		switch err {
		case ErrDraftFailed:
			continue
		case nil:
			ctx := &Subscription{
				ID: ID{
					ID:     draft.ID,
					ChatID: ctx.chat.ID,
					Source: sourceID,
				},
				Name: draft.Name,
				Item: draft.Item,
			}
			if a.Storage.Create(ctx) {
				return a.change(0, ctx.ID, Change{})
			} else {
				return errors.New("exists")
			}
		default:
			break
		}
	}
	return ErrDraftFailed
}

func (a *Aggregator) Create(tg telegram.Client, c *telegram.Command) error {
	err := a.doCreate(c)
	if err != nil {
		_, err = tg.Send(c.Chat.ID,
			&telegram.Text{Text: err.Error()},
			&telegram.SendOptions{ReplyToMessageID: c.MessageID})
	}
	return err
}

func (a *Aggregator) Resume(tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(tg, c, Change{})
}

var ErrSuspendedByUser = errors.New("suspended by user")

func (a *Aggregator) Suspend(tg telegram.Client, c *telegram.Command) error {
	return a.changeByUser(tg, c, Change{Error: ErrSuspendedByUser})
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

func (a *Aggregator) CommandListener() *telegram.CommandListener {
	return telegram.NewCommandListener().
		HandleFunc("/sub", a.Create).
		HandleFunc("resume", a.Resume).
		HandleFunc("suspend", a.Suspend).
		HandleFunc("/status", a.Status)
}