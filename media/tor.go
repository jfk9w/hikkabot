package media

import (
	"crypto/md5"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jfk9w-go/flu"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/metrics"
	"github.com/otiai10/gosseract/v2"
	"github.com/pkg/errors"
)

type Storage interface {
	FileHash(url, hash string) bool
}

type BufferSpace string

func NewBufferSpace(path string) BufferSpace {
	bs := BufferSpace(path)
	if path != "" {
		bs.Cleanup()
		if err := os.MkdirAll(path, 0644); err != nil {
			panic(err)
		}
	}

	return bs
}

func (bs BufferSpace) NewResource(size int64) Resource {
	if bs != "" {
		return NewFileResource(filepath.Join(string(bs), newID()))
	} else {
		return NewMemoryResource(int(size))
	}
}

func (bs BufferSpace) Cleanup() {
	if bs != "" {
		os.RemoveAll(string(bs))
	}
}

type Tor struct {
	metrics.Metrics
	Storage
	BufferSpace BufferSpace
	SizeBounds  [2]int64
	Buffer      bool
	Debug       bool
	Workers     int
	converters  map[string]Converter
	ocrClient   chan *gosseract.Client
	queue       chan *Promise
	work        sync.WaitGroup
}

func (tor *Tor) AddConverter(converter Converter) *Tor {
	if tor.converters == nil {
		tor.converters = make(map[string]Converter)
	}

	mimeTypes := converter.MIMETypes()
	for _, mimeType := range mimeTypes {
		if _, ok := tor.converters[mimeType]; ok {
			panic(fmt.Sprintf("MIME type %s converter already registered", mimeType))
		}

		tor.converters[mimeType] = converter
	}

	return tor
}

func (tor *Tor) Initialize() *Tor {
	if tor.Metrics == nil {
		tor.Metrics = metrics.Dummy
	}

	tor.ocrClient = make(chan *gosseract.Client, 1)
	client := gosseract.NewClient()
	tor.ocrClient <- client

	tor.queue = make(chan *Promise, tor.Workers)
	tor.work.Add(tor.Workers)
	for i := 0; i < tor.Workers; i++ {
		go func() {
			defer tor.work.Done()
			for promise := range tor.queue {
				promise.media, promise.err = tor.Materialize(promise.descriptor, promise.options)
				promise.work.Done()
			}
		}()
	}

	return tor
}

func (tor *Tor) Close() error {
	close(tor.queue)
	tor.work.Wait()
	return nil
}

func (tor *Tor) Submit(url string, descriptor Descriptor, options Options) *Promise {
	promise := &Promise{
		URL:        url,
		descriptor: descriptor,
		options:    options,
	}

	promise.work.Add(1)
	tor.queue <- promise
	return promise
}

func (tor *Tor) Materialize(descriptor Descriptor, options Options) (media Materialized, err error) {
	media, err = tor.materialize0(descriptor, options)
	if err != nil && media.Resource != nil {
		media.Resource.Cleanup()
	}

	return
}

func (tor *Tor) materialize0(descriptor Descriptor, options Options) (media Materialized, err error) {
	var metadata *Metadata
	for {
		metadata, err = descriptor.Metadata(tor.SizeBounds[1])
		if err != nil {
			err = errors.Wrap(err, "load metadata")
			return
		}

		tor.Counter("files_total", metrics.Labels{
			"mimeType": metadata.MIMEType,
		}).Inc()
		tor.Counter("files_total_size", metrics.Labels{
			"mimeType": metadata.MIMEType,
		}).Add(float64(metadata.Size))

		if metadata.Size > tor.SizeBounds[1] {
			err = errors.Errorf("exceeded hard max size %dB (%dB)",
				tor.SizeBounds[1]>>20, metadata.Size>>20)
			return
		}

		if metadata.Size < tor.SizeBounds[0] {
			err = errors.Errorf("size below threshold %dB (%dB)",
				tor.SizeBounds[0], metadata.Size)
			return
		}

		if slash := strings.Index(metadata.MIMEType, ";"); slash > 0 {
			metadata.MIMEType = metadata.MIMEType[:slash]
		}

		if options.Hashable {
			media.Resource = tor.BufferSpace.NewResource(metadata.Size)
			if err = media.Resource.Pull(descriptor); err != nil {
				err = errors.Wrap(err, "pull to hash")
				return
			}

			hash := md5.New()
			if err = flu.Copy(media.Resource, flu.Xable{W: hash}); err != nil {
				err = errors.Wrap(err, "hash")
				return
			}

			hashstr := fmt.Sprintf("%x", hash.Sum(nil))
			tor.Counter("hash_checks", metrics.Labels{
				"mimeType": metadata.MIMEType,
			}).Inc()
			if tor.FileHash(metadata.URL, hashstr) {
				log.Printf("Hash collision: %s (%s)", metadata.URL, hashstr)
				tor.Counter("hash_collisions", metrics.Labels{
					"mimeType": metadata.MIMEType,
				}).Inc()
				err = ErrFiltered
				return
			}

			descriptor = LocalDescriptor{
				Metadata_: metadata,
				Resource:  media.Resource,
			}
		}

		mimeType := metadata.MIMEType
		if mediaType, ok := MIMEType2MediaType[mimeType]; ok {
			if metadata.Size < tor.SizeBounds[0] {
				err = errors.Errorf("size below threshold %dB (%dB)",
					tor.SizeBounds[0], metadata.Size)
				return
			}

			maxSize := MaxSize(mediaType)
			if metadata.Size > maxSize[1] {
				err = errors.Errorf("exceeded max size %dMB for %s (%dMB)",
					maxSize[1]>>20, mediaType, metadata.Size>>20)
				return
			}

			media.Metadata = *metadata
			media.Type = mediaType
			if tor.Buffer || options.Buffer ||
				mediaType == telegram.Photo && (options.OCR != nil || metadata.Size > MaxSize(telegram.Photo)[0]) ||
				mediaType == telegram.Video && metadata.Size > MaxSize(telegram.Video)[0] ||
				media.Resource != nil {
				media.Resource = tor.BufferSpace.NewResource(metadata.Size)
			} else {
				media.Resource = new(VolatileResource)
			}

			if err = media.Resource.Pull(descriptor); err != nil {
				err = errors.Wrap(err, "pull descriptor")
				return
			}

			break
		}

		if converter, ok := tor.converters[mimeType]; ok {
			descriptor, err = converter.Convert(metadata, descriptor)
			if err != nil {
				err = errors.Wrapf(err, "convert via %T", converter)
				return
			}
		} else {
			err = errors.Errorf("MIME type %s is not supported", mimeType)
			return
		}
	}

	// filters
	if options.Hashable {
		if media.Type == telegram.Photo && options.OCR != nil {
			if err = tor.checkOCR(options.OCR, media); err != nil {
				if err == ErrFiltered {
					return
				} else {
					log.Printf("Failed to run OCR on image %s: %s", media.Metadata.URL, err)
					err = nil
				}
			}
		}
	}

	tor.Counter("files_materialized", metrics.Labels{
		"mimeType": metadata.MIMEType,
	}).Inc()
	tor.Counter("files_materialized_size", metrics.Labels{
		"mimeType": metadata.MIMEType,
	}).Add(float64(metadata.Size))

	return
}

var ErrFiltered = errors.New("filtered")

func (tor *Tor) checkOCR(options *OCR, media Materialized) error {
	tor.Counter("ocr_checks", metrics.Labels{
		"mimeType": media.Metadata.MIMEType,
	}).Inc()
	client := <-tor.ocrClient
	client.SetLanguage(options.Languages...)
	media.Resource.SubmitOCR(client)
	text, err := client.Text()
	tor.ocrClient <- client
	if tor.Debug {
		if err == nil {
			log.Printf("Recognized text for %s:\n%s", media.Metadata.URL, text)
		} else {
			tor.Counter("ocr_failed", metrics.Labels{
				"mimeType": media.Metadata.MIMEType,
			}).Inc()
		}
	}

	if options.Regex.MatchString(strings.ToLower(text)) {
		tor.Counter("ocr_filtered", metrics.Labels{
			"mimeType": media.Metadata.MIMEType,
		}).Inc()
		return ErrFiltered
	}

	return nil
}

var (
	symbols  = []rune("abcdefghijklmonpqrstuvwxyz0123456789")
	idLength = 16
)

func newID() string {
	id := make([]rune, idLength)
	for i := 0; i < idLength; i++ {
		id[i] = symbols[rand.Intn(len(symbols))]
	}
	return string(id)
}
