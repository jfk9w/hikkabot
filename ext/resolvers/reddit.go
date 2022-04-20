package resolvers

import (
	"context"
	"net/url"

	"hikkabot/3rdparty/reddit"
	"hikkabot/3rdparty/viddit"
	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
)

type RedditContext interface {
	reddit.Context
	viddit.Context
}

type Reddit[C RedditContext] struct {
	client httpf.Client
	viddit viddit.Interface
}

func (r *Reddit[C]) String() string {
	return "media-resolver.reddit"
}

func (r *Reddit[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var client reddit.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	var viddit viddit.Client[C]
	if err := app.Use(ctx, &viddit, false); err != nil {
		return err
	}

	r.client = client
	r.viddit = viddit
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
		url, err := r.viddit.ResolveURL(ctx, source.String())
		if err != nil {
			return nil, err
		}

		return &media.HTTPRef{
			URL:    url,
			Client: r.client,
			Buffer: true,
		}, nil

	default:
		return nil, nil
	}
}
