package core

import (
	"context"
	"os"

	"hikkabot/core/internal/blobs"
	"hikkabot/feed"
	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/pkg/errors"
)

type BlobConfig struct {
	MinSize media.Size   `yaml:"minSize,omitempty" doc:"Minimum media file size." pattern:"^(\\d+)([KMGT])?$" default:"1K"`
	MaxSize media.Size   `yaml:"maxSize,omitempty" doc:"Maximum media file size." pattern:"^(\\d+)([KMGT])?$" default:"50M"`
	TTL     flu.Duration `yaml:"ttl,omitempty" doc:"How long to keep cached files." default:"15m"`
}

type BlobContext interface {
	BlobConfig() BlobConfig
}

type Blobs[C BlobContext] struct {
	feed.Blobs
}

func (b Blobs[C]) String() string {
	return blobs.ServiceID
}

func (b *Blobs[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if b.Blobs != nil {
		return nil
	}

	dir, err := os.MkdirTemp(os.TempDir(), "blobs-")
	if err != nil {
		return errors.Wrapf(err, "create temporary directory")
	}

	config := app.Config().BlobConfig()
	blobs := &blobs.Files{
		Clock:      app,
		Dir:        dir,
		TTL:        config.TTL.Value,
		SizeBounds: [2]media.Size{config.MinSize, config.MaxSize},
	}

	if err := app.Manage(ctx, blobs); err != nil {
		return err
	}

	b.Blobs = blobs
	return nil
}

var SkipSizeCheck = blobs.SkipSizeCheck
