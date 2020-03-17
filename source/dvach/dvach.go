package dvach

import (
	"io"
	"regexp"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w/hikkabot/media"

	"github.com/jfk9w/hikkabot/api/dvach"
)

var ocr = &media.OCR{
	Languages: []string{"rus"},
	Regex: regexp.MustCompile(`(?is).*?` +
		`(т\s?в\s?о\s?я.*?м\s?а\s?(т\s?ь|м\s?а).*?(у\s?м\s?р\s?(е|ё)т|с\s?д\s?о\s?х\s?н\s?е\s?т)|` +
		`m\s?o\s?t\s?h\s?e\s?r.*?w\s?i\s?l\s?l.*?d\s?i\s?e|` +
		`п\s?р\s?о\s?к\s?л\s?я\s?т|` +
		`c\s?u\s?r\s?s\s?e).*`),
}

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
