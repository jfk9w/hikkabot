package dvach

import (
	"io"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/mediator"
)

type mediatorRequest struct {
	client *flu.Client
	file   dvach.File
}

func (r *mediatorRequest) Metadata() (*mediator.Metadata, error) {
	return &mediator.Metadata{
		URL:    r.file.URL(),
		Size:   int64(r.file.Size),
		Format: Formats[r.file.Type],
	}, nil
}

func (r *mediatorRequest) Reader() (io.Reader, error) {
	return r.client.
		GET(r.file.URL()).
		Execute().
		Reader()
}

var Formats = map[dvach.FileType]string{
	dvach.JPEG: "jpg",
	dvach.PNG:  "png",
	dvach.GIF:  "gif",
	dvach.WebM: "webm",
	dvach.MP4:  "mp4",
}
