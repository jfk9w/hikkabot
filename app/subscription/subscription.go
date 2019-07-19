package subscription

import (
	"github.com/jfk9w/hikkabot/app/media"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/pkg/errors"
)

// Context is passed to a subscription for update.
type Context struct {
	MediaManager *media.Manager
	DvachClient  *dvach.Client
	RedditClient *reddit.Client
}

// Interface is defines a subscription.
type Interface interface {

	// Service should return a human-readable description of the service this subscription is for.
	Service() string

	// Parse should try to initialize a subscription from a given input string and an optional options string.
	// It should return a hash of a subscription (in order to ensure uniqueness) and true in case of success.
	Parse(ctx Context, cmd string, opts string) (string, error)

	// Update is called when a subscription is called for update.
	Update(ctx Context, offset Offset, uc *UpdateCollection)
}

type Service = func() Interface

var (
	EmptyHash      = ""
	ErrParseFailed = errors.New("failed to parse")
)
