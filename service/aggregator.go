package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Storage interface {
	ActiveSubscribers() []telegram.ID
	InsertFeed(*Feed) bool
	NextFeed(telegram.ID) *Feed
	UpdateFeedOffset(string, int64) bool
	SuspendFeed(string, error) *Feed
}

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
				agg.suspend(feed.ID, err)
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

func (agg *Aggregator) suspend(id string, err error) {
	s := agg.storage.SuspendFeed(id, err)
	if s != nil {
		go agg.notifyAdministrators(s, fmt.Sprintf("Feed suspended. Reason: %s", err))
	}
}

func (agg *Aggregator) readOptions(rawOptions []byte, options interface{}) error {
	return json.Unmarshal(rawOptions, options)
}

func (agg *Aggregator) writeOptions(options interface{}) ([]byte, error) {
	return json.Marshal(options)
}

func (agg *Aggregator) Subscribe(chat *telegram.Chat, serviceID string, secondaryID string, name string, options interface{}) error {
	rawOptions, err := agg.writeOptions(options)
	if err != nil {
		return err
	}

	f := &Feed{
		ChatID:       chat.ID,
		ServiceID:    serviceID,
		SecondaryID:  secondaryID,
		Name:         name,
		OptionsBytes: rawOptions,
	}

	if ok := agg.storage.InsertFeed(f); !ok {
		return errors.New("exists")
	}

	go agg.notifyAdministrators(f, "Subscription OK.")
	agg.schedule(chat.ID)

	return nil
}

func (agg *Aggregator) notifyAdministrators(f *Feed, message string) {
	chat, err := agg.bot.GetChat(f.ChatID)
	if err != nil {
		log.Printf("Failed to get chat %d: %s", f.ChatID, err)
		return
	}

	var (
		chatTitle string
		adminIDs  []telegram.ID
	)

	if chat.Type == telegram.PrivateChat {
		chatTitle = "private"
		adminIDs = []telegram.ID{chat.ID}
	} else {
		chatTitle = chat.Title
		admins, err := agg.bot.GetChatAdministrators(f.ChatID)
		if err != nil {
			log.Printf("Failed to get administrator list for chat %d: %s", f.ChatID, err)
			return
		}

		adminIDs = make([]telegram.ID, 0)
		for _, admin := range admins {
			if admin.User.IsBot {
				continue
			}

			adminIDs = append(adminIDs, admin.User.ID)
		}
	}

	text := message + fmt.Sprintf(`
Chat: %s
Service: %s
Feed: #%s
`, chatTitle, f.ServiceID, f.Name)

	for _, adminID := range adminIDs {
		_, err := agg.bot.Send(adminID, text, telegram.NewSendOpts().
			Message().
			DisableWebPagePreview(true))
		if err != nil {
			log.Printf("Failed to send message to %d: %s", adminID, err)
		}
	}
}
