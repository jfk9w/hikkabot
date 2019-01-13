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

func (storage *DummyStorage) ActiveSubscribers() []telegram.ID {
	return []telegram.ID{}
}

func (storage *DummyStorage) InsertFeed(f *service.Feed) bool {
	storage.feed = f
	return true
}

func (storage *DummyStorage) NextFeed(chatID telegram.ID) *service.Feed {
	return storage.feed
}

func (storage *DummyStorage) UpdateFeedOffset(id string, offset int64) bool {
	storage.feed.Offset = offset
	return true
}

func (storage *DummyStorage) SuspendFeed(id string, err error) *service.Feed {
	s := storage.feed
	storage.feed = nil
	return s
}

func (storage *DummyStorage) InsertPostRef(pk *dvach.PostKey, ref *dvach.MessageRef) {
	storage.dvachPostHrefs[pk.String()] = ref
}

func (storage *DummyStorage) GetPostRef(pk *dvach.PostKey) (*dvach.MessageRef, bool) {
	ref, ok := storage.dvachPostHrefs[pk.String()]
	return ref, ok
}
