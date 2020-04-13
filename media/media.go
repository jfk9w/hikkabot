package media

import (
	"sync"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Metadata struct {
	URL      string
	Size     int64
	MIMEType string
}

type Options struct {
	Hashable bool
	Buffer   bool
}

type Materialized struct {
	Metadata Metadata
	Resource Resource
	Type     telegram.MediaType
}

type Promise struct {
	URL        string
	descriptor Descriptor
	options    Options
	media      Materialized
	err        error
	work       sync.WaitGroup
}

func (p *Promise) Materialize() (Materialized, error) {
	p.work.Wait()
	return p.media, p.err
}
