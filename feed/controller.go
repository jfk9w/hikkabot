package feed

import (
	"log"
	"strings"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type controller struct {
	channel  Channel
	ctx      ApplicationContext
	storage  Storage
	interval time.Duration
	active   map[telegram.ID]bool
	mu       sync.RWMutex
}

func newController(channel Channel, ctx ApplicationContext, storage Storage, interval time.Duration) *controller {
	return &controller{
		channel:  channel,
		ctx:      ctx,
		storage:  storage,
		interval: interval,
		active:   make(map[telegram.ID]bool),
	}
}

func (c *controller) init() {
	for _, chatID := range c.storage.Active() {
		c.ensure(chatID)
	}
}

func (c *controller) get(primaryID string) (*ItemData, bool) {
	return c.storage.Get(primaryID)
}

func (c *controller) run(chatID telegram.ID) {
	item, ok := c.storage.Advance(chatID)
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
	queue := &UpdateQueue{
		updates: make(chan Update, 10),
		cancel:  make(chan struct{}),
	}
	go queue.run(c.ctx, item.Offset, item)
	hasUpdates := false
	for u := range queue.updates {
		hasUpdates = true
		err := c.channel.SendUpdate(chatID, u)
		if err != nil {
			return errors.Wrapf(err, "on send update: %+v", u)
		}
		if !c.storage.Update(item.PrimaryID, Change{Offset: u.Offset}) {
			queue.cancel <- struct{}{}
			close(queue.cancel)
			return errCancelled
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, u.Offset)
			item.Offset = u.Offset
		}
	}
	if queue.err != nil {
		return errors.Wrap(queue.err, "on update")
	}
	if !hasUpdates {
		if !c.storage.Update(item.PrimaryID, Change{Offset: item.Offset}) {
			return errCancelled
		} else {
			log.Printf("Updated offset for %v: %v -> %v", item, item.Offset, item.Offset)
		}
	}
	return nil
}

func (c *controller) create(candidate Item, access *access) bool {
	item, ok := c.storage.Create(access.chat.ID, candidate)
	if ok {
		c.resume(item, access)
		return true
	}
	return false
}

func (c *controller) suspend(item *ItemData, access *access, err error) bool {
	if c.storage.Update(item.PrimaryID, Change{Error: err}) {
		log.Printf("Suspended %v: %v", item, err)
		go c.notify(item, access, &suspendEvent{err})
		return true
	}
	return false
}

func (c *controller) resume(item *ItemData, access *access) bool {
	if c.storage.Update(item.PrimaryID, Change{}) {
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
	adminIDs, err := access.getAdminIDs(c.channel)
	if err != nil {
		log.Printf("Failed to load admin IDs for %v: %v", item.ChatID, err)
	}

	var chatTitle string
	chat, _ := access.getChat(c.channel)
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
	go c.channel.SendAlert(adminIDs, sb.String(), command)
}

func (c *controller) getActiveChats() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.active)
}
