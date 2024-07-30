package reddit

import (
	"context"
)

type Interface interface {
	GetListing(ctx context.Context, subreddit, sort string, limit int) ([]Thing, error)
	GetPosts(ctx context.Context, subreddit string, ids ...string) ([]Thing, error)
	Subscribe(ctx context.Context, action SubscribeAction, subreddits []string) error
}
