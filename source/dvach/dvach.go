package dvach

import (
	"io"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w/hikkabot/media"

	"github.com/jfk9w/hikkabot/api/dvach"
)

type mediaDescriptor struct {
	client fluhttp.Client
	file   dvach.File
}

func (d *mediaDescriptor) Metadata(maxSize int64) (*media.Metadata, error) {
	return &media.Metadata{
		URL:      d.file.URL(),
		Size:     int64(d.file.Size) << 10,
		MIMEType: Type2MIMEType[d.file.Type],
	}, nil
}

func (d *mediaDescriptor) Reader() (io.Reader, error) {
	return d.client.
		GET(d.file.URL()).
		Execute().
		Reader()
}

var Type2MIMEType = map[dvach.FileType]string{
	dvach.JPEG: "image/jpeg",
	dvach.PNG:  "image/png",
	dvach.GIF:  "image/gif",
	dvach.WebM: "video/webm",
	dvach.MP4:  "video/mp4",
}
