package mediator

import (
	"context"
	"net/url"
	"sync"
	"time"

	"hikkabot/feed"
	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
)

var convertibleTypes = map[string]string{
	"video/webm": "video/mp4",
}

type Impl struct {
	Clock   syncf.Clock
	Storage feed.MediaHashStorage
	Blobs   feed.Blobs
	Locker  syncf.Locker
	Metrics me3x.Registry
	Timeout time.Duration

	resolvers  []media.Resolver
	converters []media.Converter
	ctx        context.Context
	cancel     func()
	work       syncf.WaitGroup
	once       sync.Once
}

func (m *Impl) String() string {
	return ServiceID
}

func (m *Impl) init() {
	m.ctx, m.cancel = context.WithCancel(context.Background())
}

func (m *Impl) RegisterMediaResolver(resolver media.Resolver) {
	m.resolvers = append(m.resolvers, resolver)
}

func (m *Impl) RegisterMediaConverter(converter media.Converter) {
	m.converters = append(m.converters, converter)
}

func (m *Impl) Mediate(ctx context.Context, source string, dedupKey *feed.ID) receiver.MediaRef {
	url, err := url.Parse(source)
	if err != nil {
		return receiver.MediaError{E: err}
	}

	m.once.Do(m.init)
	return syncf.AsyncWith[*receiver.Media](m.ctx, m.work.Spawn, func(ctx context.Context) (*receiver.Media, error) {
		var dedup *dedupOpts
		if dedupKey != nil {
			dedup = &dedupOpts{
				key:    *dedupKey,
				source: url,
			}
		}

		ctx, cancel := context.WithTimeout(ctx, m.Timeout)
		defer cancel()

		logf.Get(m).Tracef(ctx, "mediating [%s]", source)
		startTime := m.Clock.Now()
		media, err := m.mediate(ctx, url, dedup)
		logf.Get(m).Resultf(ctx, logf.Debug, logf.Warn,
			"mediated [%s] in %s: %v", source, m.Clock.Now().Sub(startTime), err)

		if errors.Is(err, errDuplicate) {
			return nil, nil
		}

		return media, err
	})
}

func (m *Impl) mediate(ctx context.Context, source *url.URL, dedup *dedupOpts) (*receiver.Media, error) {
	metaRef, err := m.resolve(ctx, source)
	if err != nil {
		return nil, err
	}

	meta, ref, err := m.convert(ctx, metaRef, dedup)
	if err != nil {
		return nil, err
	}

	m.incrementCounter(source, dedup, meta, err)
	mediaType := telegram.MediaTypeByMIMEType(meta.MIMEType)
	if mediaType == telegram.DefaultMediaType {
		return nil, errors.Errorf("mime type %s is not supported", meta.MIMEType)
	}

	if meta.Size > 0 && int64(meta.Size) <= mediaType.RemoteMaxSize() {
		input, err := ref.Get(ctx)
		if err != nil {
			return nil, err
		}

		return &receiver.Media{
			MIMEType: meta.MIMEType,
			Input:    input,
		}, nil
	}

	if int64(meta.Size) <= mediaType.AttachMaxSize() {
		ref := m.Blobs.Buffer(meta.MIMEType, ref)
		meta, err := ref.GetMeta(ctx)
		if err != nil {
			return nil, err
		}

		if int64(meta.Size) <= mediaType.AttachMaxSize() {
			input, err := ref.Get(ctx)
			if err != nil {
				return nil, err
			}

			return &receiver.Media{
				Input:    input,
				MIMEType: meta.MIMEType,
			}, nil
		}
	}

	return nil, errors.Errorf("size %s too large", meta.Size)
}

func (m *Impl) incrementCounter(source *url.URL, dedup *dedupOpts, meta *media.Meta, err error) {
	switch {
	case errors.Is(err, errDuplicate):
		labels := make(me3x.Labels, 0, 2).
			Add("origin", source.Host).
			Add("feed_id", dedup.key)
		m.Metrics.Counter("duplicate", labels).Inc()
	case err != nil:
		labels := make(me3x.Labels, 0, 1).
			Add("origin", source.Host)
		m.Metrics.Counter("failed", labels).Inc()
	default:
		labels := make(me3x.Labels, 0, 2).
			Add("origin", source.Host).
			Add("type", meta.MIMEType)
		m.Metrics.Counter("ok", labels).Inc()
	}

}

func (m *Impl) convert(ctx context.Context, metaRef media.MetaRef, dedup *dedupOpts) (*media.Meta, media.Ref, error) {
	meta, err := metaRef.GetMeta(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get meta")
	}

	var ref media.Ref
	if dedup != nil {
		ref = m.Blobs.Buffer(meta.MIMEType, metaRef)
		if err := m.dedup(ctx, meta.MIMEType, ref, dedup); err != nil {
			return nil, nil, err
		}
	} else {
		ref = m.bufferLeaveURL(meta.MIMEType, metaRef)
	}

	if mimeType, ok := convertibleTypes[meta.MIMEType]; ok {
		for _, converter := range m.converters {
			metaRef, err := converter.Convert(ctx, ref, mimeType)
			if metaRef == nil && err == nil {
				continue
			}

			logf.Get(m).Resultf(ctx, logf.Debug, logf.Warn, "convert with [%s]: %v", converter, err)
			if metaRef != nil {
				return m.convert(ctx, metaRef, nil)
			}
		}
	}

	return meta, ref, nil
}

func (m *Impl) dedup(ctx context.Context, mimeType string, ref media.Ref, dedup *dedupOpts) error {
	input, err := ref.Get(ctx)
	if err != nil {
		return err
	}

	now := m.Clock.Now()
	hash := &feed.MediaHash{
		FeedID:    dedup.key,
		URL:       dedup.source.String(),
		FirstSeen: now,
		LastSeen:  now,
	}

	if readImage, ok := imageTypes[mimeType]; ok {
		err = hashImage(input, hash, readImage)
	} else {
		err = hashAny(input, hash)
	}

	logf.Get(m).Resultf(ctx, logf.Debug, logf.Warn, "hash media [%s => %s]: %v", hash.URL, hash.Value, err)
	if err != nil {
		return err
	}

	ok, err := m.Storage.IsMediaUnique(ctx, hash)
	if err != nil {
		return err
	}

	if !ok {
		return errDuplicate
	}

	return nil
}

func (m *Impl) bufferLeaveURL(mimeType string, ref media.Ref) media.Ref {
	return syncf.Resolve[flu.Input](func(ctx context.Context) (flu.Input, error) {
		input, err := ref.Get(ctx)
		if err != nil {
			return nil, err
		}

		if url, ok := input.(flu.URL); ok {
			return url, nil
		}

		return m.Blobs.Buffer(mimeType, ref).Get(ctx)
	})
}

func (m *Impl) resolve(ctx context.Context, source *url.URL) (metaRef media.MetaRef, err error) {
	for _, resolver := range m.resolvers {
		metaRef, err = resolver.Resolve(ctx, source)
		if metaRef == nil && err == nil {
			continue
		}

		logf.Get(m).Resultf(ctx, logf.Debug, logf.Warn, "resolve [%s] with [%s]: %v", source, resolver, err)
		if metaRef != nil {
			logf.Get(m).Debugf(ctx, "resolve [%s] with [%s]: ok", source, resolver)
			return
		}
	}

	logf.Get(m).Debugf(ctx, "resolve [%s] as http ref", source)
	return &media.HTTPRef{URL: source.String()}, nil
}

func (m *Impl) Close() error {
	if m.cancel != nil {
		m.cancel()
		m.work.Wait()
	}

	return nil
}

type dedupOpts struct {
	key    feed.ID
	source *url.URL
}
