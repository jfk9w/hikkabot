package viddit

import "context"

type Interface interface {
	ResolveURL(ctx context.Context, url string) (string, error)
}
