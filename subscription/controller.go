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

func (c *controller) get(primaryID string) (*ItemData, bool) {
	return c.storage.GetItem(primaryID)
}

func (c *controller) run(chatID telegram.ID) {
	item, ok := c.storage.GetNextItem(chatID)
	if !ok {
		c.mu.Lock()
		delete(c.active, chatID)
		c.mu.Unlock()
		log.Printf("Stopped updater for %v", chatID)
		return
	}

	err := c.update(chatID, item)
	if err != nil {
		if err != errCancelled {
			c.suspend(item, &access{chatID: item.ChatID}, err)
		}
	}

	time.AfterFunc(c.interval, func() { c.run(chatID) })
}

var errCancelled = errors.New("cancelled")

func (c *controller) update(chatID telegram.ID, item *ItemData) error {
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

		if !c.storage.UpdateOffset(item.PrimaryID, u.Offset) {
			uc.cancel <- struct{}{}
			close(uc.cancel)
			return errCancelled
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, u.Offset)
			item.Offset = u.Offset
		}
	}

	if uc.Error != nil {
		return errors.Wrap(uc.Error, "on update")
	}

	if !hasUpdates {
		if !c.storage.UpdateOffset(item.PrimaryID, item.Offset) {
			return errCancelled
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, item.Offset)
		}
	}

	return nil
}

func (c *controller) create(candidate Item, access *access) bool {
	item, ok := c.storage.AddItem(access.chat.ID, candidate)
	if ok {
		c.resume(item, access)
		return true
	}

	return false
}

func (c *controller) suspend(item *ItemData, access *access, err error) bool {
	if c.storage.UpdateError(item.PrimaryID, err) {
		log.Printf("Suspended %v: %v", item, err)
		go c.notify(item, access, &suspendEvent{err})
		return true
	}

	return false
}

func (c *controller) resume(item *ItemData, access *access) bool {
	if c.storage.ResetError(item.PrimaryID) {
		c.ensure(item.ChatID)
		log.Printf("Resumed %v", item)
		go c.notify(item, access, resume)
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
	log.Printf("Started updater for %v", chatID)
}

func (c *controller) notify(item *ItemData, access *access, event event) {
	adminIDs, err := access.getAdminIDs(c.bot)
	if err != nil {
		log.Printf("Failed to load admin IDs for %v: %v", item.ChatID, err)
	}

	var chatTitle string
	chat, _ := access.getChat(c.bot)
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

	command := telegram.CommandButton(strings.Title(event.undo()), "/"+event.undo(), item.PrimaryID)
	for _, adminID := range adminIDs {
		go c.bot.Send(adminID,
			&telegram.Text{
				Text:                  sb.String(),
				DisableWebPagePreview: true},
			&telegram.SendOpts{
				ReplyMarkup: command})
	}
}

func (c *controller) getActiveChats() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.active)
}
