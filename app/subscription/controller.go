package subscription

import (
	"log"
	"strings"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type controller struct {
	bot      *telegram.Bot
	ctx      Context
	storage  Storage
	interval time.Duration
	active   map[telegram.ID]bool
	mu       sync.RWMutex
}

func newController(bot *telegram.Bot, ctx Context, storage Storage, interval time.Duration) *controller {
	return &controller{
		bot:      bot,
		ctx:      ctx,
		storage:  storage,
		interval: interval,
		active:   make(map[telegram.ID]bool),
	}
}

func (c *controller) init() {
	for _, chatID := range c.storage.GetActiveChats() {
		c.ensure(chatID)
	}
}

func (c *controller) get(primaryID string) (*itemData, bool) {
	return c.storage.GetItem(primaryID)
}

func (c *controller) run(chatID telegram.ID) {
	item, ok := c.storage.GetNextItem(chatID)
	if !ok {
		c.mu.Lock()
		delete(c.active, chatID)
		c.mu.Unlock()
		return
	}

	err := c.update(chatID, item)
	if err != nil {
		if err != errCancelled {
			c.suspend(item.PrimaryID, &auth{chatID: item.ChatID}, err)
		}
	}

	time.AfterFunc(c.interval, func() { c.run(chatID) })
}

var errCancelled = errors.New("cancelled")

func (c *controller) update(chatID telegram.ID, item *itemData) error {
	uc := NewUpdateCollection(10)
	go item.Update(c.ctx, item.Offset, uc)
	sender := NewSender(c.bot, chatID)
	hasUpdates := false
	for u := range uc.C {
		hasUpdates = true
		err := sender.Send(u)
		if err != nil {
			return errors.Wrap(err, "on send update")
		}

		if !c.storage.UpdateItemOffset(item.PrimaryID, u.Offset) {
			uc.cancel <- struct{}{}
			close(uc.cancel)
			return errCancelled
		}
	}

	if uc.Error != nil {
		return errors.Wrap(uc.Error, "on update")
	}

	if !hasUpdates {
		if !c.storage.UpdateItemOffset(item.PrimaryID, item.Offset) {
			return errCancelled
		}
	}

	return nil
}

func (c *controller) create(item Item, auth *auth) bool {
	primaryID, ok := c.storage.AddItem(auth.chat.ID, item)
	if ok {
		c.resume(primaryID, auth)
		return true
	}

	return false
}

func (c *controller) suspend(primaryID string, auth *auth, err error) bool {
	item, ok := c.storage.UpdateItemError(primaryID, err)
	if ok {
		go c.notify(item, auth, &suspendEvent{err})
		return true
	}

	return false
}

func (c *controller) resume(primaryID string, auth *auth) bool {
	item, ok := c.storage.UpdateItemError(primaryID, nil)
	if ok {
		c.ensure(item.ChatID)
		go c.notify(item, auth, resume)
		return true
	}

	return false
}

func (c *controller) ensure(chatID telegram.ID) {
	c.mu.RLock()
	ok := c.active[chatID]
	c.mu.RUnlock()
	if ok {
		return
	}

	c.mu.Lock()
	if c.active[chatID] {
		c.mu.Unlock()
		return
	}

	c.active[chatID] = true
	c.mu.Unlock()
	go c.run(chatID)
}

func (c *controller) notify(item *itemData, auth *auth, event event) {
	adminIDs, err := auth.getAdminIDs(c.bot)
	if err != nil {
		log.Printf("Failed to load admin IDs for %v: %v", item.ChatID, err)
	}

	var chatTitle string
	chat, _ := auth.getChat(c.bot)
	if chat.Type == telegram.PrivateChat {
		chatTitle = "<private>"
	} else {
		chatTitle = chat.Title
	}

	sb := new(strings.Builder)
	sb.WriteString("Subscription ")
	sb.WriteString(event.status())
	sb.WriteString("\nChat: ")
	sb.WriteString(chatTitle)
	sb.WriteString("\nService: ")
	sb.WriteString(item.Service())
	sb.WriteString("\nItem: ")
	sb.WriteString(item.Name())
	event.details(sb)

	command := telegram.CommandButton(strings.Title(event.undo()), event.undo(), item.PrimaryID)
	for _, adminID := range adminIDs {
		go c.bot.Send(adminID,
			&telegram.Text{
				Text:                  sb.String(),
				DisableWebPagePreview: true},
			&telegram.SendOpts{
				ReplyMarkup: command})
	}
}
