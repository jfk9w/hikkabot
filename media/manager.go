package media

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

type Config struct {
	Workers  int
	TempDir  string
	Aconvert aconvert.Config
}

func (c Config) validate() error {
	if c.Workers < 1 {
		return errors.New("there should be at least 1 worker")
	}
	os.RemoveAll(c.TempDir)
	err := os.MkdirAll(c.TempDir, os.ModeDir|os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "on temp dir creation")
	}
	return nil
}

type Manager struct {
	tempDir  string
	aconvert *aconvert.Client
	queue    chan *Media
	worker   *sync.WaitGroup
}

func NewManager(config Config, aconvertClient *aconvert.Client) *Manager {
	if err := config.validate(); err != nil {
		panic(err)
	}
	if aconvertClient == nil {
		if config.Aconvert.TestFile != "" {
			aconvertClient = aconvert.NewClient(nil, &config.Aconvert)
		} else {
			panic("no aconvert client")
		}
	}
	manager := &Manager{
		tempDir:  config.TempDir,
		aconvert: aconvertClient,
		queue:    make(chan *Media),
		worker:   new(sync.WaitGroup)}
	for i := 0; i < config.Workers; i++ {
		go manager.runDownloadWorker()
	}
	return manager
}

func (m *Manager) runDownloadWorker() {
	m.worker.Add(1)
	defer m.worker.Done()
	for media := range m.queue {
		m.downloadAndLog(media)
	}
}

func (m *Manager) Download(batch Batch) {
	for i := range batch {
		batch[i].work.Add(1)
		m.queue <- batch[i]
	}
}

func (m *Manager) Shutdown() {
	close(m.queue)
	m.worker.Wait()
	os.RemoveAll(m.tempDir)
	if m.aconvert != nil {
		m.aconvert.Shutdown()
	}
}

func (m *Manager) downloadAndLog(media *Media) {
	file := m.newTempFile()
	start := time.Now()
	type_, size, err := m.download(media.Loader, file)
	if err != nil {
		_ = os.RemoveAll(file.Path())
		media.err = err
		log.Printf("Failed to process media %s (took %v): %s", media.Href, time.Now().Sub(start), err)
	} else {
		media.file = file
		media.type_ = type_
		log.Printf("Processed media %s (size %dKb, took %v)", media.Href, size>>10, time.Now().Sub(start))
	}
	media.work.Done()
}

func (m *Manager) newTempFile() flu.File {
	path := filepath.Join(m.tempDir, ksuid.New().String())
	_ = os.RemoveAll(path)
	return flu.File(path)
}

func (m *Manager) download(loader Loader, resource flu.File) (Type, int64, error) {
	mediaType, err := loader.LoadMedia(resource)
	if err != nil {
		return 0, 0, errors.Wrap(err, "on download")
	}
	if mediaType == WebM {
		err := m.convertWebM(resource)
		if err != nil {
			return 0, 0, errors.Wrap(err, "on WebM conversion")
		}
	}
	stat, err := os.Stat(resource.Path())
	if err != nil {
		return 0, 0, errors.Wrap(err, "on stat")
	}
	size := stat.Size()
	if size > mediaType.MaxSize() {
		return 0, 0, errors.Errorf(
			"size (%d MB) exceeds limit (%d MB)",
			size>>20, mediaType.MaxSize()>>20)
	}
	if size < MinMediaSize {
		return 0, 0, errors.Errorf(
			"size (%d bytes) is below minimum size (%d bytes)",
			size, MinMediaSize)
	}
	return mediaType, size, nil
}

func (m *Manager) convertWebM(resource flu.File) error {
	r, err := m.aconvert.ConvertResource(resource, aconvert.NewOpts().TargetFormat("mp4"))
	if err != nil {
		return errors.Wrap(err, "on processing")
	}
	err = m.aconvert.Download(r, resource)
	if err != nil {
		return errors.Wrap(err, "on download")
	}
	return nil
}
