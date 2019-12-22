package media

import (
	"log"
	"sync"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"

	"github.com/pkg/errors"
)

type Config struct {
	Dir         string
	Concurrency int
	Aconvert    *aconvert.Config
}

func (c Config) validate() error {
	if c.Concurrency < 1 {
		return errors.New("concurrency should be at least 1")
	}
	return nil
}

type Manager struct {
	storage    Storage
	converters []Converter
	queue      chan *media
	workers    sync.WaitGroup
}

func NewManager(config Config) *Manager {
	err := config.validate()
	if err != nil {
		panic(err)
	}
	storage := FileStorage{config.Dir}
	err = storage.Init()
	if err != nil {
		panic(err)
	}
	manager := &Manager{
		storage:    storage,
		converters: []Converter{BaseConverter},
		queue:      make(chan *media),
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

func (m *Manager) Download(remote ...Remote) []Download {
	download := make([]Download, len(remote))
	for i, r := range remote {
		media := New(r)
		m.queue <- media
		download[i] = media
	}
	return download
}

func (m *Manager) Shutdown() {
	close(m.queue)
	m.workers.Wait()
	m.storage.Cleanup()
}

func (m *Manager) runWorker() {
	m.workers.Add(1)
	defer m.workers.Done()
	for media := range m.queue {
		res := m.storage.NewResource()
		start := time.Now()
		size, err := m.download(media, res)
		elapsed := time.Now().Sub(start)
		if err != nil {
			res.Cleanup()
			media.err = err
			log.Printf("Failed to process media %s (took %v): %s", media.URL(), elapsed, err)
		} else {
			log.Printf("Processed media %s (size %d Kb, took %v)", media.URL(), size>>10, elapsed)
		}
		media.work.Done()
	}
}

func (m *Manager) download(media *media, res ReadWrite) (int64, error) {
	typ, err := media.Download(res)
	if err != nil {
		return -1, errors.Wrap(err, "on download")
	}

loop:
	for _, converter := range m.converters {
		typ, err := converter.Convert(typ, res)
		switch err {
		case nil:
			size, err := res.Size()
			if err != nil {
				return -1, errors.Wrap(err, "on size calculation")
			}
			if size < MinMediaSize {
				return -1, errors.Errorf(
					"size (%d bytes) is below minimum size (%d bytes)",
					size, MinMediaSize)
			}
			maxSize := maxMediaSize[typ]
			if size > maxSize {
				return -1, errors.Errorf(
					"size (%d MB) exceeds limit (%d MB) for type %s",
					size>>20, maxSize>>20, typ)
			}
			media.res = res
			media.typ = typ
			return size, nil
		case UnsupportedTypeErr:
			continue loop
		default:
			return -1, errors.Wrap(err, "on conversion")
		}
	}

	return -1, UnsupportedTypeErr
}
