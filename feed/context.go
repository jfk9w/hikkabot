package feed

import (
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/media"
)

// Context is passed to a subscription for pullUpdates.
type Context struct {
	MediaManager *media.Manager
	DvachClient  *dvach.Client
	RedditClient *reddit.Client
}
