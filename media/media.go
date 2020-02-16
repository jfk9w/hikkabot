package media

import (
	"sync"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

var MIMEType2MediaType = map[string]string{
	"image/jpeg": telegram.Photo,
	"image/png":  telegram.Photo,
	"image/bmp":  telegram.Photo,
	"image/gif":  telegram.Video,
	"video/mp4":  telegram.Video,
}

type Metadata struct {
	URL      string
	Size     int64
	MIMEType string
}

type Options struct {
	Hashable bool
	OCR      *OCR
	Buffer   bool
}

type Materialized struct {
	Metadata Metadata
	Resource Resource
	Type     telegram.MediaType
}

var (
	maxPhotoSize = [2]int64{5 << 20, 10 << 20}
	maxMediaSize = [2]int64{20 << 20, 50 << 20}
)

func MaxSize(mediaType telegram.MediaType) [2]int64 {
	if mediaType == telegram.Photo {
		return maxPhotoSize
	} else {
		return maxMediaSize
	}
}

type Promise struct {
	URL        string
	descriptor Descriptor
	options    Options
	media      Materialized
	err        error
	work       sync.WaitGroup
}

func (p *Promise) Materialized() (Materialized, error) {
	p.work.Wait()
	return p.media, p.err
}
