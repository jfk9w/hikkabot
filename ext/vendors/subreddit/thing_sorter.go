package subreddit

import "github.com/jfk9w/hikkabot/3rdparty/reddit"

type thingSorter []reddit.Thing

func (ts thingSorter) Len() int {
	return len(ts)
}

func (ts thingSorter) Less(i, j int) bool {
	return ts[i].Data.ID < ts[j].Data.ID
}

func (ts thingSorter) Swap(i, j int) {
	ts[i], ts[j] = ts[j], ts[i]
}
