package subscription

import (
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/app/media"
)

// Context is passed to a subscription for update.
type Context struct {
	MediaManager *media.Manager
	DvachClient  *dvach.Client
	RedditClient *reddit.Client
}
