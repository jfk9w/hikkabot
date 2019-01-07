package service

import . "github.com/jfk9w-go/telegram-bot-api"

type Storage interface {
	Active() []ID
	Insert(*Subscription) bool
	Query(ID) *Subscription
	Update(string, Offset) bool
	Suspend(string, error) *Subscription
}

type DummyStorage struct {
	subscription *Subscription
}

func (storage *DummyStorage) Active() []ID {
	return []ID{}
}

func (storage *DummyStorage) Insert(subscription *Subscription) bool {
	storage.subscription = subscription
	return true
}

func (storage *DummyStorage) Query(chatId ID) *Subscription {
	return storage.subscription
}

func (storage *DummyStorage) Update(id string, offset Offset) bool {
	storage.subscription.Offset = offset
	return true
}

func (storage *DummyStorage) Suspend(id string, err error) *Subscription {
	subscription := storage.subscription
	storage.subscription = nil
	return subscription
}
