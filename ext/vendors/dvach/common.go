package dvach

import (
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/telegram-bot-api"

	"hikkabot/3rdparty/dvach"
	"hikkabot/core/media"
)

func NewMediaRef(httpClient *fluhttp.Client, feedID telegram.ID, file dvach.File, dedup bool) *media.Ref {
	return &media.Ref{
		Resolver: media.PlainResolver{HttpClient: httpClient},
		Metadata: &media.Metadata{
			MIMEType: file.Type.MIMEType(),
			Size:     file.Size,
		},
		URL:    file.URL(),
		Dedup:  dedup,
		FeedID: feedID,
	}
}
