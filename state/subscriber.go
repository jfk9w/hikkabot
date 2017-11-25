package state

import (
	"strings"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/telegram"
	. "github.com/jfk9w/hikkabot/util"
)

type SubscriberKey string

func getSubscriberKey(chat telegram.ChatRef) SubscriberKey {
	if len(chat.Username) > 0 {
		return SubscriberKey(chat.Username)
	} else {
		return SubscriberKey(telegram.FormatChatID(chat.ID))
	}
}

func parseSubscriberKey(key SubscriberKey) telegram.ChatRef {
	str := string(key)
	if strings.HasPrefix(str, "@") {
		return telegram.ChatRef{
			Username: str,
		}
	} else {
		return telegram.ChatRef{
			ID: telegram.ParseChatID(str),
		}
	}
}

type SubscriptionContext struct {
	Chat     telegram.ChatRef
	Board    string
	ThreadID string
	Offset   int
}

type Subscription func(SubscriptionContext)

type subscriberRT struct {
	mutex *sync.Mutex
	queue chan ThreadKey
	halt  Hook
	done  Hook
}

func (rt *subscriberRT) safe(f func()) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	f()
}

func (rt *subscriberRT) check(f func() bool) bool {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	return f()
}

type Subscriber struct {
	Active   map[ThreadKey]int            `json:"active"`
	Inactive map[ThreadKey]InactiveThread `json:"inactive"`
	rt       *subscriberRT
}

func (s *Subscriber) initRT() {
	s.rt = &subscriberRT{
		mutex: new(sync.Mutex),
		queue: make(chan ThreadKey, 20),
		halt:  NewHook(),
		done:  NewHook(),
	}
}

func (s *Subscriber) resume(key ThreadKey) {
	s.rt.safe(func() {
		if thread, ok := s.Inactive[key]; ok {
			s.Active[key] = thread.Offset
			delete(s.Inactive, key)
		} else {
			s.Active[key] = 0
		}
	})
}

func (s *Subscriber) suspend(key ThreadKey) {
	s.rt.safe(func() {
		if offset, ok := s.Active[key]; ok {
			s.Inactive[key] = newInactiveThread(offset)
			delete(s.Active, key)
		}
	})
}

func (s *Subscriber) run(subscription Subscription) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer func() {
			s.rt.done.Trigger()
			ticker.Stop()
		}()

		for {
			select {
			case <-s.rt.halt:
				return

			case <-ticker.C:
				select {
				case key := <-s.rt.queue:
					if s.rt.check(func() {
						_, ok := s.Active[key]
						return ok
					}) {
						subscription(SubscriptionContext{})
					}

				default:
				}
			}
		}
	}()
}
