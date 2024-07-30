package vendors

import (
	"github.com/jfk9w/hikkabot/v4/internal/ext/vendors/dvach"
	"github.com/jfk9w/hikkabot/v4/internal/ext/vendors/reddit"
)

type (
	SubredditConfig            = reddit.SubredditConfig
	SubredditSuggestionsConfig = reddit.SubredditSuggestionsConfig
)

func DvachCatalog[C dvach.Context]() *dvach.Catalog[C] {
	return new(dvach.Catalog[C])
}

func DvachThread[C dvach.Context]() *dvach.Thread[C] {
	return new(dvach.Thread[C])
}

func Subreddit[C reddit.SubredditContext]() *reddit.Subreddit[C] {
	return new(reddit.Subreddit[C])
}

func SubredditSuggestions[C reddit.SubredditSuggestionsContext]() *reddit.SubredditSuggestions[C] {
	return new(reddit.SubredditSuggestions[C])
}
