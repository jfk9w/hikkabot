package dvach

import (
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/media"
)

func downloadMedia(client *dvach.Client, manager *media.Manager, file dvach.File) *media.Media {
	in := &media.HTTPRequest{Request: client.NewRequest().Resource(file.URL()).GET()}
	return manager.Submit(file.URL(), Formats[file.Type], in)
}

var Formats = map[dvach.FileType]string{
	dvach.JPEG: "jpg",
	dvach.PNG:  "png",
	dvach.GIF:  "gif",
	dvach.WebM: "webm",
	dvach.MP4:  "mp4",
}
