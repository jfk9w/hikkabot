package service

import (
	"errors"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

const (
	subscriberQueueSize = 15
	subscriberTimeout   = 20 * time.Second
)

var (
	errSubscriptionExists = errors.New("subscription already exists")
)

// Subscriber is a set of active and inactive threads
type Subscriber struct {

	// Active threads are threads which are currently streaming
	Active map[ThreadKey]int `json:"active"`

	// Inactive threads are suspended threads (i.e. due to an error)
	Inactive map[ThreadKey]InactiveThread `json:"inactive"`

	_runtime subscriberRT
}

type alert func(string, string, int) (int, error)

type subscriberRT struct {
	halt  util.Hook
	done  util.Hook
	mutex *sync.RWMutex
	queue chan ThreadKey
}

func newSubscriber() *Subscriber {
	return &Subscriber{
		Active:   make(map[ThreadKey]int),
		Inactive: make(map[ThreadKey]InactiveThread),
	}
}

func (sub *Subscriber) init() {
	sub._runtime = subscriberRT{
		halt:  util.NewHook(),
		done:  util.NewHook(),
		mutex: new(sync.RWMutex),
		queue: make(chan ThreadKey, subscriberQueueSize),
	}
}

func (sub *Subscriber) halt() util.Hook {
	return sub._runtime.halt
}

func (sub *Subscriber) done() util.Hook {
	return sub._runtime.done
}

func (sub *Subscriber) mutex() *sync.RWMutex {
	return sub._runtime.mutex
}

func (sub *Subscriber) queue(key ThreadKey) {
	sub._runtime.queue <- key
}

func (sub *Subscriber) start(alert alert) {
	sub.enqueueAll()
	go func() {
		ticker := time.NewTicker(subscriberTimeout)
		defer func() {
			sub.done().Send()
			ticker.Stop()
		}()

		for {
			select {
			case <-sub.halt():
				return

			case <-ticker.C:
				select {
				case key := <-sub._runtime.queue:
					sub.next(alert, key)

				default:
				}
			}
		}
	}()
}

func (sub *Subscriber) enqueueAll() {
	sub.mutex().RLock()
	defer sub.mutex().RUnlock()

	for key := range sub.Active {
		sub.queue(key)
	}
}

func (sub *Subscriber) next(alert alert, key ThreadKey) {
	_mutex.RLock()
	defer _mutex.RUnlock()

	sub.mapActiveThread(key, func(board string, threadID string, offset int) {
		go func() {
			newOffset, err := alert(board, threadID, offset)
			_mutex.RLock()
			defer _mutex.RUnlock()
			if err == nil {
				sub.resetActiveThread(key, newOffset)
			} else {
				sub.deleteActiveThread(key)
			}
		}()
	})
}

func (sub *Subscriber) stop() {
	sub.halt().Send()
	sub.done().Wait()
}

func (sub *Subscriber) mapActiveThread(key ThreadKey, f func(string, string, int)) {
	sub.mutex().RLock()
	defer sub.mutex().RUnlock()

	if offset, ok := sub.Active[key]; ok {
		board, threadID := ParseThreadKey(key)
		f(board, threadID, offset)
	}
}

func (sub *Subscriber) resetActiveThread(key ThreadKey, offset int) {
	sub.mutex().RLock()
	defer sub.mutex().RUnlock()

	if _, ok := sub.Active[key]; ok {
		sub.Active[key] = offset
		sub.queue(key)
	}
}

func (sub *Subscriber) newActiveThread(board string, threadID string) error {
	key := FormatThreadKey(board, threadID)

	_mutex.RLock()
	defer _mutex.RUnlock()

	sub.mutex().Lock()
	defer sub.mutex().Unlock()

	if _, ok := sub.Active[key]; ok {
		return errSubscriptionExists
	}

	if inactive, ok := sub.Inactive[key]; ok {
		sub.Active[key] = inactive.Offset
		delete(sub.Inactive, key)
	} else {
		sub.Active[key] = 0
	}

	sub.queue(key)

	sawmill.Debug("thread started", sawmill.Fields{
		"key": key,
	})

	return nil
}

func (sub *Subscriber) deleteActiveThread(key ThreadKey) {
	sub.mutex().Lock()
	defer sub.mutex().Unlock()

	if offset, ok := sub.Active[key]; ok {
		sub.Inactive[key] = newInactiveThread(offset)
		delete(sub.Active, key)
		sawmill.Debug("thread stopped", sawmill.Fields{
			"key": key,
		})
	}
}

func (sub *Subscriber) deleteAllActiveThreads() {
	sub.mutex().Lock()
	defer sub.mutex().Unlock()

	for key, offset := range sub.Active {
		sub.Inactive[key] = newInactiveThread(offset)
		delete(sub.Active, key)
	}
}
