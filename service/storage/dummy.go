package storage

import (
	"github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/hikkabot/service/dvach"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type DummyStorage struct {
	feed           *service.Feed
	dvachPostHrefs map[string]*dvach.MessageRef
}

func Dummy() *DummyStorage {
	return &DummyStorage{
		dvachPostHrefs: make(map[string]*dvach.MessageRef),
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

func (s *DummyStorage) SuspendFeed(id string, err error) bool {
	if s.feed == nil {
		return false
	}

	s.feed = nil
	return true
}

func (s *DummyStorage) InsertPostRef(pk *dvach.PostKey, ref *dvach.MessageRef) {
	s.dvachPostHrefs[pk.String()] = ref
}

func (s *DummyStorage) GetPostRef(pk *dvach.PostKey) (*dvach.MessageRef, bool) {
	ref, ok := s.dvachPostHrefs[pk.String()]
	return ref, ok
}
