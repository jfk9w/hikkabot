package dvach

import "github.com/jfk9w/hikkabot/api/dvach"

type queryResults []dvach.Post

func (r queryResults) Len() int {
	return len(r)
}

func (r queryResults) Less(i, j int) bool {
	return r[i].Num < r[j].Num
}

func (r queryResults) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
