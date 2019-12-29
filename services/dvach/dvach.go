package dvach

import (
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/subscription"
)

func downloadMedia(ctx subscription.ApplicationContext, file dvach.File) *media.Media {
	in := &media.HTTPRequest{Request: ctx.DvachClient.NewRequest().Resource(file.URL()).GET()}
	return ctx.MediaManager.Submit(file.URL(), Formats[file.Type], in)
}

var Formats = map[dvach.FileType]string{
	dvach.JPEG: "jpg",
	dvach.PNG:  "png",
	dvach.GIF:  "gif",
	dvach.WebM: "webm",
	dvach.MP4:  "mp4",
}
