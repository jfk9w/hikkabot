package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

var (
	ErrInvalidFormat = errors.New("invalid format")
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("not found")
)

type Aggregator struct {
	bot      *telegram.Bot
	storage  Storage
	services map[string]Service
	subs     map[telegram.ID]struct{}
	interval time.Duration
	mu       sync.RWMutex
}

func NewAggregator(storage Storage, bot *telegram.Bot, interval time.Duration) *Aggregator {
	return &Aggregator{
		bot:      bot,
		storage:  storage,
		services: make(map[string]Service),
		subs:     make(map[telegram.ID]struct{}),
		interval: interval,
	}
}

func (agg *Aggregator) Add(services ...Service) *Aggregator {
	for _, service := range services {
		if _, ok := agg.services[service.ID()]; ok {
			panic("service " + service.ID() + " already registered")
		}

		agg.services[service.ID()] = service
	}

	return agg
}

func (agg *Aggregator) Init() *Aggregator {
	activeChatIDs := agg.storage.ActiveSubscribers()
	for _, chatID := range activeChatIDs {
		agg.subs[chatID] = struct{}{}
		go agg.run(chatID)

		log.Println("Restored active chat", chatID)
	}

	return agg
}

func (agg *Aggregator) schedule(chatID telegram.ID) {
	agg.mu.RLock()
	if _, ok := agg.subs[chatID]; ok {
		agg.mu.RUnlock()
		return
	}

	agg.mu.RUnlock()
	agg.mu.Lock()
	if _, ok := agg.subs[chatID]; ok {
		agg.mu.Unlock()
		return
	}

	agg.subs[chatID] = struct{}{}
	agg.mu.Unlock()

	go agg.run(chatID)
}

func (agg *Aggregator) cancel(chatID telegram.ID) {
	agg.mu.Lock()
	delete(agg.subs, chatID)
	agg.mu.Unlock()
}

func (agg *Aggregator) run(chatID telegram.ID) {
	feed := agg.storage.NextFeed(chatID)
	if feed == nil {
		log.Println(chatID, "has no active subscriptions")
		agg.cancel(chatID)
		return
	}

	service := agg.services[feed.ServiceID]
	updatePipe := NewUpdatePipe()
	defer updatePipe.closeOut()

	go service.Update(feed.Offset, feed.OptionsFunc(), updatePipe)

	var (
		oldOffset int64 = -1
		newOffset       = feed.Offset
	)

	for updateBatch := range updatePipe.updateCh {
		newOffset = updateBatch.offset
		updateCh := make(chan Update)
		go updateBatch.Get(updateCh)
		for update := range updateCh {
			_, err := update.Send(agg.bot, chatID)
			if err != nil {
				updatePipe.stop()
				_ = agg.set(nil, feed.ID, err)
				log.Println(feed, "suspended:", err)
				goto reschedule
			}
		}

		if newOffset != oldOffset {
			if oldOffset != -1 {
				log.Println(feed, oldOffset, "->", newOffset)
				if ok := agg.storage.UpdateFeedOffset(feed.ID, oldOffset); !ok {
					log.Println(feed, "interrupted")
					updatePipe.stop()
					goto reschedule
				}
			}

			oldOffset = newOffset
		}
	}

	log.Println(feed, oldOffset, "->", newOffset)
	_ = agg.storage.UpdateFeedOffset(feed.ID, newOffset)

reschedule:
	time.AfterFunc(agg.interval, func() { agg.run(chatID) })
}

func (agg *Aggregator) set(userID *telegram.ID, id string, err error) error {
	feed := agg.storage.GetFeed(id)
	if feed == nil {
		return ErrNotFound
	}

	var chat *EnrichedChat
	if userID != nil {
		var err error
		chat, err = agg.enrichChat(feed.ChatID, nil)
		if err != nil {
			return err
		}

		if !chat.hasAdminAccess(*userID) {
			return ErrForbidden
		}
	}

	var ok bool
	if err == nil {
		ok = agg.storage.ResumeFeed(id)
	} else {
		ok = agg.storage.SuspendFeed(id, err)
	}

	if !ok {
		return ErrNotFound
	}

	if userID == nil {
		var err error
		chat, err = agg.enrichChat(feed.ChatID, nil)
		if err != nil {
			return err // whatever
		}
	}

	var (
		text   string
		markup telegram.ReplyMarkup
	)

	if err == nil {
		text = fmt.Sprintf(`Subscription resumed.
Chat: %s
Service: %s
Title: #%s`, chat.title, feed.ServiceID, feed.Name)
		markup = telegram.CommandButton("Suspend", "/suspend", feed.ID)
	} else {
		text = fmt.Sprintf(`Subscription suspended.
Chat: %s
Service: %s
Title: #%s
Reason: %s`, chat.title, feed.ServiceID, feed.Name, err)
		markup = telegram.CommandButton("Resume", "/resume", feed.ID)
	}

	go chat.forEachAdminID(func(adminID telegram.ID) {
		_, err := agg.bot.Send(adminID, text, telegram.NewSendOpts().Message().ReplyMarkup(markup))
		if err != nil {
			log.Printf("Failed to send message to %d: %s", adminID, err)
		}
	})

	if err == nil {
		agg.schedule(feed.ChatID)
	}

	return nil
}

func (agg *Aggregator) readOptions(rawOptions []byte, options interface{}) error {
	return json.Unmarshal(rawOptions, options)
}

func (agg *Aggregator) writeOptions(options interface{}) ([]byte, error) {
	return json.Marshal(options)
}

func (agg *Aggregator) Subscribe(chat *EnrichedChat, serviceID string, secondaryID string, name string, options interface{}) error {
	rawOptions, err := agg.writeOptions(options)
	if err != nil {
		return err
	}

	feed := &Feed{
		ChatID:       chat.ID,
		ServiceID:    serviceID,
		SecondaryID:  secondaryID,
		Name:         name,
		OptionsBytes: rawOptions,
	}

	if ok := agg.storage.InsertFeed(feed); !ok {
		return errors.New("exists")
	}

	go chat.forEachAdminID(func(adminID telegram.ID) {
		_, err := agg.bot.Send(
			adminID,
			fmt.Sprintf(`Subscription OK.
Chat: %s
Service: %s
Title: #%s`, chat.title, serviceID, name),
			telegram.NewSendOpts().
				Message().
				ReplyMarkup(telegram.CommandButton("Suspend", "/suspend", feed.ID)))
		if err != nil {
			log.Printf("Failed to send message to %s: %s", adminID, err)
		}
	})

	agg.schedule(chat.ID)
	return nil
}

func (agg *Aggregator) SubscribeCommandListener(c *telegram.Command) {
	if c.Payload == "" {
		c.ErrorReply(ErrInvalidFormat)
		return
	}

	tokens := strings.Split(c.Payload, " ")

	var (
		chatID telegram.ChatID
		chat   *telegram.Chat
	)

	if len(tokens) > 1 && tokens[1] != "" && tokens[1] != "." {
		chatID = telegram.Username(tokens[1])
	} else {
		chat = c.Chat
	}

	enriched, err := agg.enrichChat(chatID, chat)
	if err != nil {
		c.ErrorReply(err)
		return
	}

	if !enriched.hasAdminAccess(c.User.ID) {
		c.ErrorReply(ErrForbidden)
		return
	}

	var options string
	if len(tokens) > 2 {
		options = tokens[2]
	}

	for _, svc := range agg.services {
		err = svc.Subscribe(tokens[0], enriched, options)
		switch err {
		case nil:
			return

		case ErrInvalidFormat:
			continue

		default:
			c.ErrorReply(err)
			return
		}
	}

	c.ErrorReply(ErrInvalidFormat)
}

func (agg *Aggregator) SuspendCommandListener(c *telegram.Command) {
	if err := agg.set(&c.User.ID, c.Payload, errors.New("suspended by user")); err != nil {
		c.ErrorReply(err)
	} else {
		c.TextReply("OK")
	}
}

func (agg *Aggregator) ResumeCommandListener(c *telegram.Command) {
	if err := agg.set(&c.User.ID, c.Payload, nil); err != nil {
		c.ErrorReply(err)
	} else {
		c.TextReply("OK")
	}
}

func (agg *Aggregator) enrichChat(chatID telegram.ChatID, chat *telegram.Chat) (*EnrichedChat, error) {
	if chat == nil {
		var err error
		chat, err = agg.bot.GetChat(chatID)
		if err != nil {
			return nil, err
		}
	} else {
		chatID = chat.ID
	}

	enriched := &EnrichedChat{Chat: chat}
	if chat.Type == telegram.PrivateChat {
		enriched.title = "private"
		enriched.adminIDs = []telegram.ID{chat.ID}
	} else {
		admins, err := agg.bot.GetChatAdministrators(chatID)
		if err != nil {
			return nil, err
		}

		adminIDs := make([]telegram.ID, 0)
		for _, admin := range admins {
			if admin.User.IsBot {
				continue
			}

			adminIDs = append(adminIDs, admin.User.ID)
		}

		enriched.title = chat.Title
		enriched.adminIDs = adminIDs
	}

	return enriched, nil
}

type EnrichedChat struct {
	*telegram.Chat
	adminIDs []telegram.ID
	title    string
}

func (c *EnrichedChat) hasAdminAccess(userID telegram.ID) bool {
	for _, adminID := range c.adminIDs {
		if adminID == userID {
			return true
		}
	}

	return false
}

func (c *EnrichedChat) forEachAdminID(f func(telegram.ID)) {
	for _, adminID := range c.adminIDs {
		f(adminID)
	}
}
