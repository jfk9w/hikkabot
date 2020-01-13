package mediator

import (
	"expvar"
	"log"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Mediator struct {
	convs   []Converter
	queue   chan *Future
	work    sync.WaitGroup
	minSize int64
	maxSize int64
	metrics *expvar.Map
	buffer  bool
}

func New(config Config) *Mediator {
	if config.Concurrency < 1 {
		panic("concurrency should be greater than 0")
	}
	mediator := &Mediator{
		convs:   []Converter{SupportedFormats},
		queue:   make(chan *Future),
		metrics: expvar.NewMap("media"),
		buffer:  config.Buffer,
	}
	mediator.minSize = config.MinSize.Value(2 << 10)
	mediator.maxSize = config.MaxSize.Value(75 << 20)
	for i := 0; i < config.Concurrency; i++ {
		go mediator.runWorker()
	}
	return mediator
}

func (m *Mediator) AddConverter(conv Converter) *Mediator {
	m.convs = append(m.convs, conv)
	return m
}

func (m *Mediator) Submit(url string, req Request) *Future {
	future := &Future{
		URL: url,
		req: req,
	}
	future.work.Add(1)
	m.queue <- future
	return future
}

func (m *Mediator) Shutdown() {
	close(m.queue)
	m.work.Wait()
}

func (m *Mediator) runWorker() {
	m.work.Add(1)
	defer m.work.Done()
	for future := range m.queue {
		future.res, future.err = m.process(future.URL, future.req)
		future.work.Done()
	}
}

func (m *Mediator) process(url string, req Request) (*telegram.Media, error) {
	start := time.Now()
	meta, err := req.Metadata()
	if err != nil {
		return nil, errors.Wrap(err, "get metadata")
	}
	if meta.Size > m.maxSize {
		return nil, errors.Errorf("size (%d MB) exceeds hard limit (%d MB)", meta.Size>>20, m.maxSize>>20)
	}
	for _, conv := range m.convs {
		creq, err := conv.Convert(req, meta)
		switch err {
		case nil:
			m.metrics.Add("size", meta.Size)
			m.metrics.Add("files", 1)
			cmeta, err := creq.Metadata()
			if err != nil {
				return nil, errors.Wrap(err, "get converted metadata")
			}
			csize, typ := cmeta.Size, creq.MediaType
			if csize < m.minSize {
				return nil, errors.Errorf("size (%d B) is below minimum size (%d B)", csize, m.minSize)
			}
			maxSize := MaxSize(typ)
			if csize > maxSize[1] {
				return nil, errors.Errorf("size (%d MB) exceeds limit (%d MB) for type %s", csize>>20, maxSize[1]>>20, typ)
			}
			media := &telegram.Media{Type: typ, URL: cmeta.URL}
			if csize > maxSize[0] || cmeta.ForceLoad {
				var in flu.Readable
				if m.buffer {
					buf := flu.NewBuffer()
					if err = flu.Copy(req, buf); err != nil {
						return nil, err
					}
					in = buf
				} else {
					in = req
				}
				media.Resource = in
			}
			log.Printf("Processed %s %s (%d KB) via %T in %v: %+v", typ, url, csize>>10, conv, time.Now().Sub(start), media)
			if _, ok := conv.(FormatSupport); !ok {
				m.metrics.Add("size", csize)
				m.metrics.Add("files", 1)
			}
			return media, nil
		case ErrUnsupportedType:
			continue
		default:
			return nil, errors.Wrap(err, "conversion failed")
		}
	}
	return nil, ErrUnsupportedType
}
