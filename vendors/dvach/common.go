package dvach

import (
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/feed"
)

func NewMediaRef(httpClient *fluhttp.Client, feedID telegram.ID, file dvach.File, dedup bool) *feed.MediaRef {
	return &feed.MediaRef{
		MediaResolver: feed.DummyMediaResolver{HttpClient: httpClient},
		URL:           file.URL(),
		Dedup:         dedup,
		FeedID:        feedID,
	}
}
