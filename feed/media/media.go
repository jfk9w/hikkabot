package media

import (
	"context"
	"net/url"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/syncf"
)

type Blob interface {
	flu.Input
	flu.Output
}

type Meta struct {
	MIMEType string
	Size     Size
}

type Ref = syncf.Ref[flu.Input]

type MetaRef interface {
	Ref
	GetMeta(ctx context.Context) (*Meta, error)
}

type Resolver interface {
	String() string
	Resolve(ctx context.Context, source *url.URL) (MetaRef, error)
}

type Converter interface {
	String() string
	Convert(ctx context.Context, ref Ref, targetType string) (MetaRef, error)
}
