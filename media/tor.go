package media

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/rivo/duplo"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/metrics"
	"github.com/otiai10/gosseract/v2"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

type Tor struct {
	metrics.Metrics
	Directory  string
	SizeBounds [2]int64
	Buffer     bool
	Debug      bool
	ImgHashes  *duplo.Store
	Workers    int
	converters map[string]Converter
	ocrClient  chan *gosseract.Client
	queue      chan *Promise
	work       sync.WaitGroup
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
				startTime := time.Now()
				promise.media, promise.err = tor.Materialize(promise.descriptor, promise.options)
				tor.Counter("materialize_time", nil).Add(time.Now().Sub(startTime).Seconds())
				tor.Counter("materialize_items", nil).Inc()
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
		metadata, err = descriptor.Metadata()
		if err != nil {
			err = errors.Wrap(err, "load metadata")
			return
		}

		if metadata.Size > tor.SizeBounds[1] {
			err = errors.Errorf("exceeded hard max size %dB (%dB)",
				tor.SizeBounds[1]>>20, metadata.Size>>20)
			return
		}

		if slash := strings.Index(metadata.MIMEType, ";"); slash > 0 {
			metadata.MIMEType = metadata.MIMEType[:slash]
		}

		mimeType := metadata.MIMEType
		if mediaType, ok := MIMEType2MediaType[mimeType]; ok {
			if metadata.Size < tor.SizeBounds[0] {
				err = errors.Errorf("size below threshold %dB for %s (%dB)",
					tor.SizeBounds[0], mediaType, metadata.Size)
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
			media.Resource = tor.newResource(metadata.Size, mediaType, options)
			if err = media.Resource.Pull(descriptor); err != nil {
				err = errors.Wrap(err, "pull descriptor")
				return
			}

			break
		}

		if converter, ok := tor.converters[mimeType]; ok {
			descriptor, err = converter.Convert(mimeType, descriptor)
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
	if media.Type == telegram.Photo {
		if ocr := options.OCR; ocr != nil {
			if err = tor.checkOCR(ocr, media); err != nil {
				if err == ErrFiltered {
					return
				} else {
					log.Printf("Failed to run OCR on image %s: %s", media.Metadata.URL, err)
					err = nil
				}
			}
		}

		if options.Hashable && tor.ImgHashes != nil {
			if err = tor.checkImageHash(media); err != nil {
				if err == ErrFiltered {
					return
				} else {
					log.Printf("Failed to hash check image %s: %s", media.Metadata.URL, err)
					err = nil
				}
			}
		}
	}

	return
}

var (
	ErrUnsupportedImage = errors.New("unsupported image")
)

const MinImageDiffScore = 2

func (tor *Tor) checkImageHash(media Materialized) error {
	var decoder func(io.Reader) (image.Image, error)
	switch media.Metadata.MIMEType {
	case "image/jpeg":
		decoder = jpeg.Decode
	case "image/png":
		decoder = png.Decode
	case "image/bmp":
		decoder = bmp.Decode
	default:
		return ErrUnsupportedImage
	}

	r, err := media.Resource.Reader()
	if err != nil {
		return errors.Wrap(err, "read")
	}

	img, err := decoder(r)
	if err != nil {
		return errors.Wrap(err, "decode")
	}

	hash, _ := duplo.CreateHash(img)
	for _, match := range tor.ImgHashes.Query(hash) {
		if match.Score < MinImageDiffScore {
			if tor.Debug {
				log.Printf("Image %s is similar to image %s (distance %.2f)",
					media.Metadata.URL, match.ID, match.Score)
				return ErrFiltered
			}
		}
	}

	tor.ImgHashes.Add(media.Metadata.URL, hash)
	return nil
}

var ErrFiltered = errors.New("filtered")

func (tor *Tor) checkOCR(options *OCR, media Materialized) error {
	client := <-tor.ocrClient
	client.SetLanguage(options.Languages...)
	media.Resource.SubmitOCR(client)
	text, err := client.Text()
	tor.ocrClient <- client
	if tor.Debug {
		if err == nil {
			log.Printf("Recognized text for %s:\n%s", media.Metadata.URL, text)
		}
	}

	if options.Regex.MatchString(strings.ToLower(text)) {
		return ErrFiltered
	}

	return nil
}

func (tor *Tor) newResource(size int64, mediaType telegram.MediaType, options Options) Resource {
	if tor.Buffer || options.Buffer ||
		(mediaType == telegram.Video && size > MaxSize(telegram.Video)[0]) ||
		(mediaType == telegram.Photo && (size > MaxSize(telegram.Photo)[0] || options.Hashable || options.OCR != nil)) {

		if tor.Directory != "" {
			return NewFileResource(tor.Directory, newID())
		} else {
			return NewMemoryResource(int(size))
		}
	} else {
		return new(VolatileResource)
	}
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
