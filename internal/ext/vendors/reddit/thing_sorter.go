package reddit

import "github.com/jfk9w/hikkabot/v4/internal/3rdparty/reddit"

type thingSorter []reddit.Thing

func (ts thingSorter) Len() int {
	return len(ts)
}

func (ts thingSorter) Less(i, j int) bool {
	return ts[i].Data.NumID < ts[j].Data.NumID
}

func (ts thingSorter) Swap(i, j int) {
	ts[i], ts[j] = ts[j], ts[i]
}
