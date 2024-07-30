package resolvers

import (
	"context"
	"net/url"
	"strings"

	"github.com/jfk9w/hikkabot/internal/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/internal/feed/media"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
)

type Dvach[C dvach.Context] struct {
	client httpf.Client
}

func (r *Dvach[C]) String() string {
	return "media-resolver.dvach"
}

func (r *Dvach[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var client dvach.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	r.client = &client
	return nil
}

func (r *Dvach[C]) Resolve(ctx context.Context, url *url.URL) (media.MetaRef, error) {
	if !strings.Contains(url.Host, "2ch.hk") {
		return nil, nil
	}

	return &media.HTTPRef{
		URL:    url.String(),
		Client: r.client,
		Buffer: true,
	}, nil
}
