package dvach

import (
	"io"
	"regexp"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/mediator"
)

var (
	ocrre = regexp.MustCompile(`(?is).*?(твоя.*?ма(ть|ма).*?(умр(е|ё)т|сдохнет)|mother.*?will.*?die|проклят|curse).*`)
	ocr   = mediator.OCR{
		Filtered:  true,
		Languages: []string{"rus", "eng"},
		Regexp:    ocrre,
	}
)

type mediatorRequest struct {
	client *flu.Client
	file   dvach.File
}

func (r *mediatorRequest) Metadata() (*mediator.Metadata, error) {
	return &mediator.Metadata{
		URL:    r.file.URL(),
		Size:   int64(r.file.Size) << 10,
		Format: Formats[r.file.Type],
		OCR:    ocr,
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
