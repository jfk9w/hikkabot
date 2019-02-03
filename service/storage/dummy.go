package storage

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/service"
)

type DummyStorage struct {
	feed        *service.Feed
	messageRefs map[string]service.MessageRef
}

func Dummy() *DummyStorage {
	return &DummyStorage{
		messageRefs: make(map[string]service.MessageRef),
	}
}

func (s *DummyStorage) ActiveSubscribers() []telegram.ID {
	return []telegram.ID{}
}

func (s *DummyStorage) InsertFeed(f *service.Feed) bool {
	s.feed = f
	return true
}

func (s *DummyStorage) NextFeed(chatID telegram.ID) *service.Feed {
	return s.feed
}

func (s *DummyStorage) UpdateFeedOffset(id string, offset int64) bool {
	s.feed.Offset = offset
	return true
}

func (s *DummyStorage) GetFeed(id string) *service.Feed {
	return s.feed
}

func (s *DummyStorage) ResumeFeed(id string) bool {
	return false
}

func (s *DummyStorage) SuspendFeed(id string, err error) bool {
	if s.feed == nil {
		return false
	}

	s.feed = nil
	return true
}

func (s *DummyStorage) StoreMessage(chatID telegram.ID, serviceID service.ID, key service.MessageKey, ref service.MessageRef) {
	s.messageRefs[key.String()] = ref
}

func (s *DummyStorage) GetMessage(chatID telegram.ID, serviceID service.ID, key service.MessageKey) (*service.MessageRef, bool) {
	ref, ok := s.messageRefs[key.String()]
	return &ref, ok
}
