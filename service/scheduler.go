package service

import (
	"log"
	"sync"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Scheduler struct {
	storage  Storage
	b        *telegram.Bot
	services map[ServiceType]SubscribeService
	tasks    map[telegram.ID]struct{}
	interval time.Duration
	mu       sync.RWMutex
}

func NewScheduler(storage Storage, b *telegram.Bot, interval time.Duration) *Scheduler {
	return &Scheduler{
		storage:  storage,
		b:        b,
		services: make(map[ServiceType]SubscribeService),
		tasks:    make(map[telegram.ID]struct{}),
		interval: interval,
	}
}

func (scheduler *Scheduler) Register(services ...SubscribeService) *Scheduler {
	for _, service := range services {
		if _, ok := scheduler.services[service.ServiceType()]; ok {
			panic("service " + service.ServiceType() + " already registered")
		}

		scheduler.services[service.ServiceType()] = service
	}

	return scheduler
}

func (scheduler *Scheduler) Init() *Scheduler {
	activeChatIDs := scheduler.storage.Active()
	for _, chatID := range activeChatIDs {
		scheduler.Schedule(chatID)
	}

	log.Printf("Restored %d active chats", len(activeChatIDs))
	return scheduler
}

func (scheduler *Scheduler) Schedule(chatID telegram.ID) {
	scheduler.mu.RLock()
	if _, ok := scheduler.tasks[chatID]; ok {
		scheduler.mu.RUnlock()
		return
	}

	scheduler.mu.RUnlock()
	scheduler.mu.Lock()
	if _, ok := scheduler.tasks[chatID]; ok {
		scheduler.mu.Unlock()
		return
	}

	scheduler.tasks[chatID] = struct{}{}
	scheduler.mu.Unlock()

	go scheduler.run(chatID)
}

func (scheduler *Scheduler) Cancel(chatID telegram.ID) {
	scheduler.mu.Lock()
	delete(scheduler.tasks, chatID)
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) run(chatID telegram.ID) {
	log.Printf("Running scheduled task for %s\n", chatID.StringValue())

	subscription := scheduler.storage.Query(chatID)
	if subscription == nil {
		log.Printf("No active subscriptions found for %s\n", chatID.StringValue())
		return
	}

	log.Println("Loaded", subscription)

	service := scheduler.services[subscription.Type]
	feed := NewFeed(subscription.Name)
	defer feed.CloseOut()

	go service.Update(subscription.Offset, subscription.Options, feed)

	var (
		oldOffset = Offset(-1)
		newOffset = subscription.Offset
	)

	for update := range feed.C {
		newOffset = update.Offset
		err := update.Send(scheduler.b, chatID)
		if err != nil {
			feed.Interrupt()
			service.Suspend(subscription.ID, err)
			log.Println("Suspending", subscription, "due to", err)
			return
		}

		if newOffset != oldOffset {
			if oldOffset != -1 {
				log.Println("Updating", subscription, "offset from", oldOffset, "to", newOffset)
				if ok := scheduler.storage.Update(subscription.ID, oldOffset); !ok {
					feed.Interrupt()
					return
				}
			}

			oldOffset = newOffset
		}
	}

	_ = scheduler.storage.Update(subscription.ID, newOffset)
	time.AfterFunc(scheduler.interval, func() {
		scheduler.run(chatID)
	})

	log.Println("Task for", chatID, "rescheduled")
}
