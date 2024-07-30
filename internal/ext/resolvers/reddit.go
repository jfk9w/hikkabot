package resolvers

import (
	"context"
	"net/url"

	"github.com/jfk9w/hikkabot/internal/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/internal/3rdparty/redditsave"
	"github.com/jfk9w/hikkabot/internal/feed/media"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

type RedditContext interface {
	reddit.Context
	redditsave.Context
}

type Reddit[C RedditContext] struct {
	client     httpf.Client
	redditsave redditsave.Interface
}

func (r *Reddit[C]) String() string {
	return "media-resolver.reddit"
}

func (r *Reddit[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var client reddit.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	var redditsave redditsave.Client[C]
	if err := app.Use(ctx, &redditsave, false); err != nil {
		return err
	}

	r.client = client
	r.redditsave = redditsave
	return nil
}

func (r *Reddit[C]) Resolve(ctx context.Context, source *url.URL) (media.MetaRef, error) {
	switch source.Host {
	case "preview.redd.it":
		return &media.HTTPRef{
			URL:    source.String(),
			Client: r.client,
			Buffer: true,
		}, nil

	case "v.redd.it":
		url, err := r.redditsave.ResolveURL(ctx, source.String())
		if err != nil {
			return nil, errors.Wrap(err, "via redditsave")
		}

		return &media.HTTPRef{
			URL:    url,
			Client: r.client,
			Buffer: true,
			Meta: &media.Meta{
				MIMEType: "video/mp4",
			},
		}, nil

	default:
		return nil, nil
	}
}
