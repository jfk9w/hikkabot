package dvach

import (
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/telegram-bot-api/feed"
)

func newMediaRef(client *fluhttp.Client, feedID feed.ID, file File, dedup bool) *feed.MediaRef {
	return &feed.MediaRef{
		MediaResolver: feed.DummyMediaResolver{Client: client},
		URL:           file.URL(),
		Dedup:         dedup,
		FeedID:        feedID,
	}
}
