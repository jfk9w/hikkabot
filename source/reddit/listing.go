package reddit

import "github.com/jfk9w/hikkabot/api/reddit"

type listing []reddit.Thing

func (t listing) Len() int {
	return len(t)
}

func (t listing) Less(i, j int) bool {
	return t[i].Data.Created.Before(t[j].Data.Created)
}

func (t listing) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
