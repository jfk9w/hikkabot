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
	Workers int
	TempDir string
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
	tempDir        string
	aconvertClient *aconvert.Client
	queue          chan *Media
	wg             *sync.WaitGroup
}

func NewManager(config Config, aconvertClient *aconvert.Client) *Manager {
	if err := config.validate(); err != nil {
		panic(err)
	}

	m := &Manager{
		tempDir:        config.TempDir,
		aconvertClient: aconvertClient,
		queue:          make(chan *Media),
		wg:             new(sync.WaitGroup)}

	for i := 0; i < config.Workers; i++ {
		go m.runDownloadWorker()
	}

	return m
}

func (m *Manager) runDownloadWorker() {
	m.wg.Add(1)
	defer m.wg.Done()
	for media := range m.queue {
		m.downloadAndLog(media)
	}
}

func (m *Manager) Download(media []Media) {
	for i := range media {
		m.queue <- (&media[i]).init()
	}
}

func (m *Manager) Shutdown() {
	close(m.queue)
	m.wg.Wait()
	os.RemoveAll(m.tempDir)
}

func (m *Manager) downloadAndLog(media *Media) {
	resource := m.newTempResource()
	startTime := time.Now()
	mediaType, size, err := m.download(media.Factory, resource)
	if err != nil {
		_ = os.RemoveAll(resource.Path())
		media.err = err
		log.Printf("Failed to process media %s (took %v): %s", media.Href, time.Now().Sub(startTime), err)
	} else {
		media.resource = resource
		media.mediaType = mediaType
		log.Printf("Processed media %s (size %dKb, took %v)", media.Href, size>>10, time.Now().Sub(startTime))
	}

	media.complete()
}

func (m *Manager) newTempResource() flu.FileSystemResource {
	path := filepath.Join(m.tempDir, ksuid.New().String())
	_ = os.RemoveAll(path)
	return flu.NewFileSystemResource(path)
}

func (m *Manager) download(factory Factory, resource flu.FileSystemResource) (Type, int64, error) {
	mediaType, err := factory(resource)
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

func (m *Manager) convertWebM(resource flu.FileSystemResource) error {
	r, err := m.aconvertClient.ConvertResource(resource, aconvert.NewOpts().TargetFormat("mp4"))
	if err != nil {
		return errors.Wrap(err, "on processing")
	}

	err = m.aconvertClient.Download(r, resource)
	if err != nil {
		return errors.Wrap(err, "on download")
	}

	return nil
}
