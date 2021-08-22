package feed

import (
	"context"
	"sync"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/ext/blob"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"

	"github.com/sirupsen/logrus"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type (
	MediaResolver interface {
		GetClient() *fluhttp.Client
		ResolveURL(ctx context.Context, client *fluhttp.Client, url string, maxSize int64) (string, error)
	}

	MediaConverter interface {
		MIMETypes() []string
		Convert(ctx context.Context, ref *MediaRef) (richtext.MediaRef, error)
	}

	MediaDedup interface {
		Check(ctx context.Context, feedID telegram.ID, url, mimeType string, blob blob.Blob) error
	}
)

type DummyMediaResolver struct {
	HttpClient *fluhttp.Client
}

func (r DummyMediaResolver) GetClient() *fluhttp.Client {
	return r.HttpClient
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
	Storage       blob.Storage
	Converters    map[string]MediaConverter
	Dedup         MediaDedup
	RateLimiter   flu.RateLimiter
	Metrics       metrics.Registry
	Retries       int
	CURL          string
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

func (m *MediaManager) Submit(ref *MediaRef) richtext.MediaRef {
	m.work.Add(1)
	mvar := richtext.NewMediaVar()
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Minute)
	ref.Manager = m
	go func() {
		defer m.work.Done()
		defer cancel()
		if err := m.RateLimiter.Start(ctx); err != nil {
			logrus.WithField("media", ref.URL).Warnf("failed to process: %s", err)
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
	FeedID      telegram.ID
	ResolvedURL string
	MediaMetadata
}

func (r *MediaRef) getClient() *fluhttp.Client {
	if r.GetClient() != nil {
		return r.GetClient()
	} else {
		return r.Manager.DefaultClient
	}
}

func (r *MediaRef) incrementMediaMethod(mimeType string, method string) {
	r.Manager.Metrics.Counter("ok", metrics.Labels{
		"feed_id":   r.FeedID,
		"mime_type": mimeType,
		"method":    method,
	}).Inc()
}

func (r *MediaRef) incrementMediaError(mimeType string, err string) {
	r.Manager.Metrics.Counter("err", metrics.Labels{
		"feed_id":   r.FeedID,
		"mime_type": mimeType,
		"err":       err,
	}).Inc()
}

func (r *MediaRef) Get(ctx context.Context) (*richtext.Media, error) {
	media, err := r.doGet(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "with resolved URL %s", r.ResolvedURL)
	}

	return media, nil
}

func (r *MediaRef) doGet(ctx context.Context) (*richtext.Media, error) {
	var err error
	r.ResolvedURL, err = r.ResolveURL(ctx, r.getClient(), r.URL, telegram.Video.AttachMaxSize())
	if err != nil {
		r.incrementMediaError("unknown", "resolve url")
		return nil, errors.Wrapf(err, "resolve url: %s", r.URL)
	}

	client := &DefaultMediaClient{r.getClient(), r.Manager.Retries}
	if r.MIMEType == "" && r.Size == 0 {
		if m, err := client.Metadata(ctx, r.ResolvedURL); err != nil {
			r.incrementMediaError("unknown", "head")
			return nil, errors.Wrap(err, "head")
		} else {
			r.MediaMetadata = *m
		}

		if r.Size != UnknownSize {
			if r.Size < r.Manager.SizeBounds[0] {
				r.incrementMediaError(r.MIMEType, "too small")
				return nil, errors.Errorf("size of %db is too low", r.Size)
			} else if r.Size > r.Manager.SizeBounds[1] {
				r.incrementMediaError(r.MIMEType, "too large")
				return nil, errors.Errorf("size %dMb too large", r.Size>>20)
			}
		}
	}

	mimeType := r.MIMEType
	if converter, ok := r.Manager.Converters[mimeType]; ok {
		ref, err := converter.Convert(ctx, r)
		if err != nil {
			r.incrementMediaError(r.MIMEType, "convert")
			return nil, errors.Wrapf(err, "convert from %s", mimeType)
		}

		return ref.Get(ctx)
	}

	mediaType := telegram.MediaTypeByMIMEType(mimeType)
	if mediaType == telegram.DefaultMediaType {
		r.incrementMediaError(r.MIMEType, "mime")
		return nil, errors.Errorf("unsupported mime type: %s", mimeType)
	}

	if r.Size != UnknownSize && r.Size <= mediaType.RemoteMaxSize() && !r.Dedup && !r.Blob {
		r.incrementMediaMethod(r.MIMEType, "remote")
		return &richtext.Media{
			MIMEType: mimeType,
			Input:    flu.URL(r.ResolvedURL),
		}, nil
	}

	if r.Size == UnknownSize || r.Size <= mediaType.AttachMaxSize() {
		blob, err := r.Manager.Storage.Alloc()
		if err != nil {
			return nil, errors.Wrap(err, "create blob")
		}

		counter := &flu.IOCounter{Output: blob}
		if err := client.Contents(ctx, r.ResolvedURL, counter); err != nil {
			r.incrementMediaError(r.MIMEType, "download")
			return nil, errors.Wrap(err, "download")
		}

		if counter.Value() <= mediaType.AttachMaxSize() {
			if r.Dedup {
				if err := r.Manager.Dedup.Check(ctx, r.FeedID, r.URL, mimeType, blob); err != nil {
					r.incrementMediaError(r.MIMEType, "dedup")
					return nil, err
				}
			}

			r.incrementMediaMethod(r.MIMEType, "attach")
			return &richtext.Media{
				MIMEType: mimeType,
				Input:    blob,
			}, nil
		}
	}

	r.incrementMediaError(r.MIMEType, "too large")
	return nil, errors.Errorf("size %dMb is too large", r.Size>>20)
}
