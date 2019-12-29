package feed

import (
	"expvar"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Aggregator struct {
	Channel
	Context
	Storage
	Timeout  time.Duration
	Aliases  map[telegram.Username]telegram.ID
	Services []Service
	chats    map[telegram.ID]bool
	mu       sync.RWMutex
}

func (a *Aggregator) Init() *Aggregator {
	a.chats = make(map[telegram.ID]bool)
	for _, chatID := range a.Active() {
		a.RunFeed(chatID)
	}
	return a
}

func (a *Aggregator) runUpdater(chatID telegram.ID) {
	item := a.Advance(chatID)
	if item == nil {
		a.mu.Lock()
		delete(a.chats, chatID)
		a.mu.Unlock()
		log.Printf("Stopped updater for %v", chatID)
		return
	}

	err := a.pullUpdates(chatID, item)
	if err != nil {
		if err != ErrNotFound {
			a.Update(0, item.PrimaryID, Change{Error: err})
		}
	}

	time.AfterFunc(a.Timeout, func() { a.runUpdater(chatID) })
}

func (a *Aggregator) pullUpdates(chatID telegram.ID, item *ItemData) error {
	queue := newUpdateQueue()
	go queue.pull(a.Context, item.Offset, item)
	hasUpdates := false
	for update := range queue.updates {
		hasUpdates = true
		err := a.SendUpdate(chatID, update)
		if err != nil {
			return errors.Wrapf(err, "on send pullUpdates: %+v", update)
		}
		expvar.Get("sent_updates").(*expvar.Int).Add(1)
		err = a.Update(0, item.PrimaryID, Change{Offset: update.Offset})
		if err != nil {
			queue.cancel <- struct{}{}
			close(queue.cancel)
			return err
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, update.Offset)
			item.Offset = update.Offset
		}
	}
	if queue.err != nil {
		return errors.Wrap(queue.err, "pull updates")
	}
	if !hasUpdates {
		if err := a.Update(0, item.PrimaryID, Change{Offset: item.Offset}); err != nil {
			return err
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, item.Offset)
		}
	}
	return nil
}

func (a *Aggregator) RunFeed(chatID telegram.ID) {
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
	go a.runUpdater(chatID)
	log.Printf("Started updater for %v", chatID)
}

func (a *Aggregator) notify(item *ItemData, change Change, ctx *changeContext) {
	adminIDs, err := ctx.getAdminIDs(a)
	if err != nil {
		return
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
	text := fmt.Sprintf(
		"Subscription %s\nChat: %s\nService: %s\nItem: %s",
		status, title, item.Service(), item.Name())
	var button telegram.ReplyMarkup
	if change.Error != nil {
		button = telegram.CommandButton("Resume", "resume", item.PrimaryID)
		text += fmt.Sprintf("\nReason: %s", change.Error.Error())
	} else {
		button = telegram.CommandButton("Suspend", "suspend", item.PrimaryID)
	}
	go a.SendAlert(adminIDs, text, button)
}

var ErrNotFound = errors.New("not found")

func (a *Aggregator) Update(userID telegram.ID, id string, change Change) (err error) {
	var item *ItemData
	var ctx *changeContext
	if userID != 0 {
		item = a.Get(id)
		if item == nil {
			return ErrNotFound
		}
		ctx = &changeContext{chatID: item.ChatID}
		err = ctx.checkAccess(a, userID)
		if err != nil {
			return
		}
	}
	ok := a.Storage.Update(id, change)
	if !ok {
		err = ErrNotFound
		return
	}
	if change.Offset != 0 {
		return
	}
	if ctx == nil {
		item = a.Get(id)
		if item == nil {
			err = ErrNotFound
			return
		}
		ctx = &changeContext{chatID: item.ChatID}
	}
	if change.Error == nil {
		chat, err := ctx.getChat(a)
		if err != nil {
			return err
		}
		a.RunFeed(chat.ID)
	}
	a.notify(item, change, ctx)
	return
}

func (a *Aggregator) changeByUser(tg telegram.Client, c *telegram.Command, change Change) error {
	reply := "OK"
	if err := a.Update(c.User.ID, c.Payload, change); err != nil {
		reply = err.Error()
	}
	_, err := tg.AnswerCallbackQuery(c.CallbackQueryID,
		&telegram.AnswerCallbackQueryOptions{Text: reply})
	return err
}

func (a *Aggregator) doCreate(c *telegram.Command) (err error) {
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
	err = ctx.checkAccess(a, c.User.ID)
	if err != nil {
		return
	}
	options := ""
	if len(fields) > 2 {
		options = fields[2]
	}
	for _, service := range a.Services {
		item := service()
		err = item.Parse(a.Context, cmd, options)
		switch err {
		case ErrParseFailed:
			continue
		case nil:
			idata := a.Storage.Create(ctx.chat.ID, item)
			if idata != nil {
				err = a.Update(0, idata.PrimaryID, Change{})
			} else {
				err = errors.New("exists")
			}
			return
		default:
			break
		}
	}
	err = ErrParseFailed
	return
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
	a.mu.RLock()
	count := len(a.chats)
	a.mu.RUnlock()
	_, err := tg.Send(c.Chat.ID,
		&telegram.Text{Text: fmt.Sprintf("OK. Active chats: %d", count)},
		&telegram.SendOptions{ReplyToMessageID: c.MessageID})
	return err
}

func (a *Aggregator) CommandListener() *telegram.CommandListener {
	return telegram.NewCommandListener().
		HandleFunc("/sub", a.Create).
		HandleFunc("resume", a.Resume).
		HandleFunc("suspend", a.Suspend).
		HandleFunc("/status", a.Status)
}
