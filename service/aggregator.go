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
	bot         *telegram.Bot
	storage     Storage
	messages    MessageStorage
	services    map[ID]Service
	activeChats map[telegram.ID]struct{}
	interval    time.Duration
	aliases     map[telegram.Username]telegram.ID
	mu          sync.RWMutex
}

func NewAggregator(bot *telegram.Bot, storage Storage, interval time.Duration, aliases map[telegram.Username]telegram.ID) *Aggregator {
	agg := &Aggregator{
		bot:         bot,
		storage:     storage,
		services:    make(map[ID]Service),
		activeChats: make(map[telegram.ID]struct{}),
		interval:    interval,
		aliases:     aliases,
	}

	if messages, ok := storage.(MessageStorage); ok {
		agg.messages = messages
	}

	return agg
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
		agg.activeChats[chatID] = struct{}{}
		go agg.run(chatID)

		log.Println("Restored active chat", chatID)
	}

	return agg
}

func (agg *Aggregator) schedule(chatID telegram.ID) {
	agg.mu.RLock()
	if _, ok := agg.activeChats[chatID]; ok {
		agg.mu.RUnlock()
		return
	}

	agg.mu.RUnlock()
	agg.mu.Lock()
	if _, ok := agg.activeChats[chatID]; ok {
		agg.mu.Unlock()
		return
	}

	agg.activeChats[chatID] = struct{}{}
	agg.mu.Unlock()

	go agg.run(chatID)
	log.Println("Scheduled", chatID)
}

func (agg *Aggregator) cancel(chatID telegram.ID) {
	agg.mu.Lock()
	delete(agg.activeChats, chatID)
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
	pipe := NewUpdatePipe()
	defer pipe.closeOut()

	go service.Update(feed.Offset, feed.OptionsFunc(), pipe)

	var (
		oldOffset int64 = -1
		newOffset       = feed.Offset
	)

	gmf := agg.gmf(feed)
	for update := range pipe.updateCh {
		newOffset = update.Offset
		m, err := update.Send(agg.bot, gmf)
		if err != nil {
			pipe.stop()

			_ = agg.set(nil, feed.ID, err)
			log.Println(feed, "suspended:", err)
			goto reschedule
		}

		if agg.messages != nil && update.Key != nil &&
			m.Chat.Type == telegram.Channel && m.Chat.Username != nil {
			agg.messages.StoreMessage(chatID, feed.ServiceID, update.Key, MessageRef{*m.Chat.Username, m.ID})
		}

		if newOffset != oldOffset {
			if oldOffset != -1 {
				log.Println(feed, oldOffset, "->", newOffset)
				if ok := agg.storage.UpdateFeedOffset(feed.ID, oldOffset); !ok {
					log.Println(feed, "interrupted")
					pipe.stop()
					goto reschedule
				}
			}

			oldOffset = newOffset
		}
	}

	if pipe.Err != nil {
		log.Println(feed, oldOffset, "->", pipe.Err)
		_ = agg.set(nil, feed.ID, pipe.Err)
	} else {
		log.Println(feed, oldOffset, "->", newOffset)
		_ = agg.storage.UpdateFeedOffset(feed.ID, newOffset)
	}

reschedule:
	time.AfterFunc(agg.interval, func() { agg.run(chatID) })
}

func (agg *Aggregator) gmf(feed *Feed) GetMessageFunc {
	if agg.messages == nil {
		return func(k MessageKey) (*MessageRef, bool) {
			return nil, false
		}
	} else {
		return func(k MessageKey) (*MessageRef, bool) {
			return agg.messages.GetMessage(feed.ChatID, feed.ServiceID, k)
		}
	}
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
			return err
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

func (agg *Aggregator) Subscribe(chat *EnrichedChat, serviceID ID, secondaryID string, name string, options interface{}) error {
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

	text := fmt.Sprintf(`Subscription OK.
Chat: %s
Service: %s
Title: #%s`, chat.title, serviceID, name)

	opts := telegram.NewSendOpts().
		Message().
		ReplyMarkup(telegram.CommandButton("Suspend", "/suspend", feed.ID))

	go chat.forEachAdminID(func(adminID telegram.ID) {
		_, err := agg.bot.Send(adminID, text, opts)
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
		if username, ok := chatID.(telegram.Username); ok {
			if unaliased, ok := agg.aliases[username]; ok {
				chatID = unaliased
			}
		}

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
