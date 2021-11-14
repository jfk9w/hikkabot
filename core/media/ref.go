package media

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Ref struct {
	*Context
	Resolver
	*Metadata
	FeedID      telegram.ID
	URL         string
	ResolvedURL string
	Dedup       bool
	Blob        bool
}

func (r *Ref) Labels() me3x.Labels {
	return r.labels(true)
}

func (r *Ref) labels(forLog bool) me3x.Labels {
	labels := me3x.Labels{}.
		Add("feed", r.FeedID)

	if forLog {
		labels = labels.Add("url", r.URL)
		if r.ResolvedURL != "" && r.ResolvedURL != r.URL {
			labels = labels.Add("resolved", r.ResolvedURL)
		}
	}

	if r.Metadata != nil {
		labels = labels.Add("type", r.MIMEType)
	} else if !forLog {
		labels = labels.Add("type", "unknown")
	}

	return labels
}

func (r *Ref) Get(ctx context.Context) (*media.Value, error) {
	media, err := r.doGet(ctx)
	if err != nil {
		logrus.WithFields(r.labels(true).Map()).Warnf("media: %s", err)
		r.Counter("failed", r.labels(false)).Inc()
		return nil, err
	} else if media == nil {
		logrus.WithFields(r.labels(true).Map()).Info("media: duplicate")
		r.Counter("duplicate", r.labels(false)).Inc()
		return nil, nil
	} else {
		logrus.WithFields(r.labels(true).Map()).Debug("media: ok")
		r.Counter("ok", r.labels(false)).Inc()
		return media, nil
	}
}

func (r *Ref) doGet(ctx context.Context) (*media.Value, error) {
	httpClient := r.GetClient(r.HttpClient)

	var err error
	r.ResolvedURL, err = r.Resolve(ctx, httpClient, r.URL, telegram.Video.AttachMaxSize())
	if err != nil {
		return nil, err
	}

	downloader := &downloader{httpClient, r.Retries}
	if r.Metadata == nil {
		if m, err := downloader.DownloadMetadata(ctx, r.ResolvedURL); err != nil {
			return nil, err
		} else {
			r.Metadata = m
		}

		if r.Size > 0 {
			if r.Size < r.SizeBounds[0] {
				return nil, errors.Errorf("size of %db is too low", r.Size)
			} else if r.Size > r.SizeBounds[1] {
				return nil, errors.Errorf("size %dMb too large", r.Size>>20)
			}
		}
	}

	mimeType := r.MIMEType
	if converter, ok := r.Converters[mimeType]; ok {
		ref, err := converter.Convert(ctx, r)
		if err != nil {
			return nil, errors.Wrapf(err, "%T", converter)
		}

		return ref.Get(ctx)
	}

	mediaType := telegram.MediaTypeByMIMEType(mimeType)
	if mediaType == telegram.DefaultMediaType {
		return nil, errors.New("unsupported mime type")
	}

	if r.Size > 0 && r.Size <= mediaType.RemoteMaxSize() && !r.Dedup && !r.Blob {
		return &media.Value{
			MIMEType: mimeType,
			Input:    flu.URL(r.ResolvedURL),
		}, nil
	}

	if r.Size <= mediaType.AttachMaxSize() {
		blob, err := r.Alloc(r.Now())
		if err != nil {
			return nil, err
		}

		counter := &flu.IOCounter{Output: blob}
		if err := downloader.DownloadContents(ctx, r.ResolvedURL, counter); err != nil {
			return nil, err
		}

		if counter.Value() <= mediaType.AttachMaxSize() {
			if r.Dedup {
				if ok, err := r.Check(ctx, r.FeedID, r.URL, mimeType, blob); err != nil || !ok {
					return nil, err
				}
			}

			return &media.Value{
				MIMEType: mimeType,
				Input:    blob,
			}, nil
		}
	}

	return nil, errors.Errorf("size %dMb is too large", r.Size>>20)
}
