package media

import (
	"log"
	"sync"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"

	"github.com/pkg/errors"
)

type Config struct {
	Concurrency int
	Aconvert    *aconvert.Config
}

type Manager struct {
	converters []Converter
	queue      chan *Media
	workers    sync.WaitGroup
}

func NewManager(config Config) *Manager {
	if config.Concurrency < 1 {
		panic("concurrency should be greater than 0")
	}
	manager := &Manager{
		converters: []Converter{SupportedFormats},
		queue:      make(chan *Media),
	}
	if config.Aconvert != nil {
		aconverter := NewAconverter(*config.Aconvert)
		manager.AddConverter(aconverter)
	}
	for i := 0; i < config.Concurrency; i++ {
		go manager.runWorker()
	}
	return manager
}

func (m *Manager) AddConverter(converter Converter) *Manager {
	m.converters = append(m.converters, converter)
	return m
}

func (m *Manager) Submit(media *Media) {
	media.work.Add(1)
	m.queue <- media
}

func (m *Manager) Shutdown() {
	close(m.queue)
	m.workers.Wait()
}

func (m *Manager) runWorker() {
	m.workers.Add(1)
	defer m.workers.Done()
	for media := range m.queue {
		err := m.download(media)
		if err != nil {
			log.Printf("Failed to process media %s: %s", media.URL, err)
		}
		media.err = err
		media.work.Done()
	}
}

func (m *Manager) download(media *Media) error {
	start := time.Now()
	for _, converter := range m.converters {
		typ, err := converter.Convert(media)
		switch err {
		case nil:
			size, err := media.in.Size()
			if err != nil {
				return errors.Wrap(err, "size calculation")
			}
			if size < MinMediaSize {
				return errors.Errorf("size (%d bytes) is below minimum size (%d bytes)", size, MinMediaSize)
			}
			if maxSize, ok := MaxMediaSize[typ]; ok && size > maxSize {
				return errors.Errorf("size (%d MB) exceeds limit (%d MB) for type %s", size>>20, maxSize>>20, typ)
			}
			media.ready = &TypeAwareReadable{Readable: media.in, Type: typ}
			log.Printf("Processed %s %s (%d Kb) via %T in %v", typ, media.URL, size>>10, converter, time.Now().Sub(start))
			return nil
		case UnsupportedTypeErr:
			continue
		default:
			return errors.Wrap(err, "conversion failed")
		}
	}
	return UnsupportedTypeErr
}
