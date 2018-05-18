package stat

import (
	"sort"
	"time"

	"github.com/jfk9w-go/dvach"
)

type Thread struct {
	dvach.Post
	PostsPerHour float64
}

type ThreadList []Thread

func (tl ThreadList) Len() int {
	return len(tl)
}

func (tl ThreadList) Less(i, j int) bool {
	return tl[i].PostsPerHour > tl[j].PostsPerHour
}

func (tl ThreadList) Swap(i, j int) {
	tl[i], tl[j] = tl[j], tl[i]
}

func Top(posts []dvach.Post) []Thread {
	tl := make([]Thread, 0)
	now := time.Now()
	for _, post := range posts {
		age := now.Sub(post.Date()).Hours()
		if age < 0.1 {
			continue
		}

		pph := float64(post.PostsCount) / age
		tl = append(tl, Thread{post, pph})
	}

	sort.Sort(ThreadList(tl))
	return tl
}
