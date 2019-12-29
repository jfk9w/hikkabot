package media

import (
	"expvar"
	"log"
	"sync"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"

	"github.com/pkg/errors"
)

type Size struct {
	Bytes     int64
	Kilobytes int64
	Megabytes int64
}

func (s *Size) Value(defaultValue int64) int64 {
	if s == nil {
		return defaultValue
	} else {
		return s.Megabytes<<20 + s.Kilobytes<<10 + s.Bytes
	}
}

type Config struct {
	Concurrency      int
	Aconvert         *aconvert.Config
	MinSize, MaxSize *Size
}

type Manager struct {
	convs   []Converter
	queue   chan *Media
	work    sync.WaitGroup
	minSize int64
	maxSize int64
}

func NewManager(config Config) *Manager {
	if config.Concurrency < 1 {
		panic("concurrency should be greater than 0")
	}
	manager := &Manager{
		convs: []Converter{SupportedFormats},
		queue: make(chan *Media),
	}
	if config.Aconvert != nil {
		aconverter := NewAconverter(*config.Aconvert)
		manager.AddConverter(aconverter)
	}
	manager.minSize = config.MinSize.Value(2 << 10)
	manager.maxSize = config.MaxSize.Value(75 << 20)
	for i := 0; i < config.Concurrency; i++ {
		go manager.runWorker()
	}
	return manager
}

func (m *Manager) AddConverter(conv Converter) *Manager {
	m.convs = append(m.convs, conv)
	return m
}

var ErrNotLoaded = errors.New("not loaded")

func (m *Manager) Submit(url string, format string, in SizeAwareReadable) *Media {
	media := &Media{
		URL:    url,
		format: format,
		in:     in,
		err:    ErrNotLoaded,
	}
	if media.in != nil {
		media.work.Add(1)
		m.queue <- media
	}
	return media
}

func (m *Manager) Shutdown() {
	close(m.queue)
	m.work.Wait()
}

func (m *Manager) runWorker() {
	m.work.Add(1)
	defer m.work.Done()
	for media := range m.queue {
		media.out, media.err = m.process(media.URL, media.format, media.in)
		media.work.Done()
	}
}

func (m *Manager) process(url string, format string, in SizeAwareReadable) (*TypeAwareReadable, error) {
	start := time.Now()
	for _, conv := range m.convs {
		in, typ, err := conv.Convert(format, in)
		switch err {
		case nil:
			size, err := in.Size()
			if err != nil {
				return nil, errors.Wrap(err, "size calculation")
			}
			if size < m.minSize {
				return nil, errors.Errorf("size (%d B) is below minimum size (%d B)", size, m.minSize)
			}
			if size > m.maxSize {
				return nil, errors.Errorf("size (%d MB) exceeds hard limit (%d MB)", size>>20, m.maxSize>>20)
			}
			if maxSize, ok := MaxMediaSize[typ]; ok && size > maxSize {
				return nil, errors.Errorf("size (%d MB) exceeds limit (%d MB) for type %s", size>>20, maxSize>>20, typ)
			}
			log.Printf("Processed %s %s (%d KB) via %T in %v", typ, url, size>>10, conv, time.Now().Sub(start))
			expvar.Get("processed_media_bytes").(*expvar.Int).Add(size)
			expvar.Get("processed_media_files").(*expvar.Int).Add(1)
			return &TypeAwareReadable{Readable: in, Type: typ}, nil
		case UnsupportedTypeErr:
			continue
		default:
			return nil, errors.Wrap(err, "conversion failed")
		}
	}
	return nil, UnsupportedTypeErr
}
