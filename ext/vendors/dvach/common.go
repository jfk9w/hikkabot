package dvach

import (
	"github.com/jfk9w-go/flu/httpf"

	"github.com/jfk9w-go/telegram-bot-api"

	"hikkabot/3rdparty/dvach"
	"hikkabot/core/media"
)

func NewMediaRef(client httpf.Client, feedID telegram.ID, file dvach.File, dedup bool) *media.Ref {
	return &media.Ref{
		Resolver: media.PlainResolver{client},
		Metadata: &media.Metadata{
			MIMEType: file.Type.MIMEType(),
			Size:     file.Size,
		},
		URL:    file.URL(),
		Dedup:  dedup,
		FeedID: feedID,
	}
}
