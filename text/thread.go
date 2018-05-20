package text

import (
	"fmt"
	"time"

	"github.com/jfk9w-go/dvach"
)

type Thread struct {
	*dvach.Thread
	PostsPerHour float64
}

func enrichThreads(threads []*dvach.Thread) []Thread {
	now := time.Now()
	r := make([]Thread, 0)
	for _, thread := range threads {
		date, ok := dvach.ToTime(thread.DateString)
		if !ok {
			continue
		}

		age := now.Sub(date).Hours()
		r = append(r, Thread{thread, float64(thread.PostsCount) / age})
	}

	return r
}

func FormatThread(thread Thread) string {
	chunks := format(thread.Item, 275)
	if len(chunks) == 0 {
		return ""
	}

	header := fmt.Sprintf("<b>%s</b>\n%s\n%d ps, %.2f ps/h\n---\n",
		thread.DateString, FormatRef(thread.Ref), thread.PostsCount, thread.PostsPerHour)

	return header + chunks[0]
}
