package dvach

import (
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/media"
)

type defaultMediaLoader struct {
	file   *dvach.File
	client *dvach.Client
}

func (l defaultMediaLoader) LoadMedia(resource flu.ResourceWriter) (media.Type, error) {
	var type_ media.Type
	switch l.file.Type {
	case dvach.WebM:
		type_ = media.WebM
	case dvach.MP4, dvach.GIF:
		type_ = media.Video
	default:
		type_ = media.Photo
	}
	return type_, l.client.DownloadFile(l.file, resource)
}
