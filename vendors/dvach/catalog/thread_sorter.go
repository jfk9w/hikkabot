package catalog

import "github.com/jfk9w/hikkabot/3rdparty/dvach"

type threadSorter []dvach.Post

func (ts threadSorter) Len() int {
	return len(ts)
}

func (ts threadSorter) Less(i, j int) bool {
	return ts[i].Num < ts[j].Num
}

func (ts threadSorter) Swap(i, j int) {
	ts[i], ts[j] = ts[j], ts[i]
}
