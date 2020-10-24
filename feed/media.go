package feed

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"mime"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type (
	MediaResolver interface {
		GetClient() *fluhttp.Client
		ResolveURL(ctx context.Context, client *fluhttp.Client, url string, maxSize int64) (string, error)
		Request(request *fluhttp.Request) *fluhttp.Request
	}

	MediaConverter interface {
		MIMETypes() []string
		Convert(ctx context.Context, ref *MediaRef) (format.MediaRef, error)
	}

	MediaDedup interface {
		Check(ctx context.Context, feedID ID, url string, blob format.Blob) error
	}
)

type DummyMediaResolver struct {
	Client *fluhttp.Client
}

func (r DummyMediaResolver) GetClient() *fluhttp.Client {
	return r.Client
}

func (r DummyMediaResolver) ResolveURL(_ context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return url, nil
}

func (r DummyMediaResolver) Request(request *fluhttp.Request) *fluhttp.Request {
	return request
}

type MediaManager struct {
	DefaultClient *fluhttp.Client
	SizeBounds    [2]int64
	Storage       format.Blobs
	Converters    map[string]MediaConverter
	Dedup         MediaDedup
	RateLimiter   flu.RateLimiter
	Metrics       metrics.Registry
	Retries       int
	ctx           context.Context
	cancel        func()
	work          sync.WaitGroup
}

func (m *MediaManager) Init(ctx context.Context) *MediaManager {
	if m.Metrics == nil {
		m.Metrics = metrics.DummyRegistry{}
	}

	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancel = cancel
	return m
}

func (m *MediaManager) Submit(ref *MediaRef) format.MediaRef {
	m.work.Add(1)
	mvar := format.NewMediaVar()
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Minute)
	ref.Manager = m
	go func() {
		defer m.work.Done()
		defer cancel()
		if err := m.RateLimiter.Start(ctx); err != nil {
			log.Printf("[media > %s] failed to process: %s", ref.URL, err)
			return
		}

		defer m.RateLimiter.Complete()
		media, err := ref.Get(ctx)
		mvar.Set(media, err)
	}()

	return mvar
}

func (m *MediaManager) Converter(converter MediaConverter) *MediaManager {
	if m.Converters == nil {
		m.Converters = map[string]MediaConverter{}
	}

	for _, mimeType := range converter.MIMETypes() {
		m.Converters[mimeType] = converter
	}

	return m
}

func (m *MediaManager) Close() {
	m.cancel()
	m.work.Wait()
}

const UnknownSize int64 = -1

type MediaRef struct {
	MediaResolver
	Manager     *MediaManager
	URL         string
	Dedup       bool
	Blob        bool
	FeedID      ID
	ResolvedURL string
	MIMEType    string
	Size        int64
}

func (r *MediaRef) getClient() *fluhttp.Client {
	if r.GetClient() != nil {
		return r.GetClient()
	} else {
		return r.Manager.DefaultClient
	}
}

func (r *MediaRef) Handle(resp *http.Response) error {
	var err error
	r.MIMEType, _, err = mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return errors.Wrapf(err, "failed to parse Content-Type: %s", resp.Header.Get("Content-Type"))
	}

	r.Size, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		r.Size = UnknownSize
	}

	return nil
}

func (r *MediaRef) incrementMediaMethod(mimeType string, method string) {
	r.Manager.Metrics.Counter("ok", metrics.Labels{
		"feed_id", PrintID(r.FeedID),
		"mime_type", mimeType,
		"method", method,
	}).Inc()
}

func (r *MediaRef) incrementMediaError(mimeType string, err string) {
	r.Manager.Metrics.Counter("err", metrics.Labels{
		"feed_id", PrintID(r.FeedID),
		"mime_type", mimeType,
		"err", err,
	}).Inc()
}

func (r *MediaRef) Get(ctx context.Context) (format.Media, error) {
	var err error
	r.ResolvedURL, err = r.ResolveURL(ctx, r.getClient(), r.URL, telegram.Video.AttachMaxSize())
	if err != nil {
		r.incrementMediaError("unknown", "resolve url")
		return format.Media{}, errors.Wrapf(err, "resolve url: %s", r.URL)
	}

	if r.MIMEType == "" && r.Size == 0 {
		if err := r.Request(r.getClient().HEAD(r.ResolvedURL)).
			Context(ctx).
			Execute().
			HandleResponse(r).
			Error; err != nil {
			r.incrementMediaError("unknown", "head")
			return format.Media{}, errors.Wrap(err, "head")
		}

		if r.Size != UnknownSize {
			if r.Size < r.Manager.SizeBounds[0] {
				r.incrementMediaError(r.MIMEType, "too small")
				return format.Media{}, errors.Errorf("size of %db is too low", r.Size)
			} else if r.Size > r.Manager.SizeBounds[1] {
				r.incrementMediaError(r.MIMEType, "too large")
				return format.Media{}, errors.Errorf("size %dMb too large", r.Size>>20)
			}
		}
	}

	mimeType := r.MIMEType
	if converter, ok := r.Manager.Converters[mimeType]; ok {
		ref, err := converter.Convert(ctx, r)
		if err != nil {
			r.incrementMediaError(r.MIMEType, "convert")
			return format.Media{}, errors.Wrapf(err, "convert from %s", mimeType)
		}

		return ref.Get(ctx)
	}

	mediaType := telegram.MediaTypeByMIMEType(mimeType)
	if mediaType == telegram.DefaultMediaType {
		r.incrementMediaError(r.MIMEType, "mime")
		return format.Media{}, errors.Errorf("unsupported mime type: %s", mimeType)
	}

	if r.Size != UnknownSize && r.Size <= mediaType.RemoteMaxSize() && !r.Dedup && !r.Blob {
		r.incrementMediaMethod(r.MIMEType, "remote")
		return format.Media{
			MIMEType: mimeType,
			Input:    flu.URL(r.ResolvedURL),
		}, nil
	}

	if r.Size == UnknownSize || r.Size <= mediaType.AttachMaxSize() {
		blob, err := r.Manager.Storage.Alloc()
		if err != nil {
			return format.Media{}, errors.Wrap(err, "create blob")
		}

		counter := &flu.IOCounter{Output: blob}
		var downloadErr error
		for i := 0; i <= r.Manager.Retries; i++ {
			if downloadErr = r.Request(r.getClient().GET(r.ResolvedURL)).
				Context(ctx).
				Execute().
				CheckStatus(http.StatusOK).
				DecodeBodyTo(counter).
				Error; downloadErr != nil {
				counter.Add(-counter.Value())
				select {
				case <-ctx.Done():
					return format.Media{}, ctx.Err()
				case <-time.After(time.Duration(5*i*i) * time.Second):
					continue
				}
			} else {
				break
			}
		}

		if downloadErr != nil {
			r.incrementMediaError(r.MIMEType, "download")
			return format.Media{}, errors.Wrap(err, "download")
		}

		if counter.Value() <= mediaType.AttachMaxSize() {
			if r.Dedup {
				if err := r.Manager.Dedup.Check(ctx, r.FeedID, r.URL, blob); err != nil {
					r.incrementMediaError(r.MIMEType, "dedup")
					log.Printf("[media > %d > %s] failed dedup check: %s", r.FeedID, r.URL, err)
					return format.Media{}, err
				}
			}

			r.incrementMediaMethod(r.MIMEType, "attach")
			return format.Media{
				MIMEType: mimeType,
				Input:    blob,
			}, nil
		}
	}

	r.incrementMediaError(r.MIMEType, "too large")
	return format.Media{}, errors.Errorf("size %dMb is too large", r.Size>>20)
}

type MD5MediaDedup struct {
	Hashes Hashes
}

func (d MD5MediaDedup) Check(ctx context.Context, feedID ID, url string, blob format.Blob) error {
	hash := md5.New()
	if err := flu.Copy(blob, flu.IO{W: hash}); err != nil {
		return errors.Wrap(err, "hash")
	}
	return d.Hashes.Check(ctx, feedID, url, fmt.Sprintf("%x", hash.Sum(nil)))
}
