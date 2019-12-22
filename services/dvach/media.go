package dvach

import (
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/media"
)

type Media struct {
	file   *dvach.File
	client *dvach.Client
}

func (m Media) URL() string {
	return dvach.Host + m.file.Path
}

func (m Media) Download(out flu.Writable) (media.Type, error) {
	typ, ok := mediaTypes[m.file.Type]
	if !ok {
		typ = "jpg"
	}
	return typ, m.client.DownloadFile(m.file, out)
}

var mediaTypes = map[dvach.FileType]media.Type{
	dvach.JPEG: "jpg",
	dvach.PNG:  "png",
	dvach.GIF:  "gif",
	dvach.WebM: "webm",
	dvach.MP4:  "mp4",
}
