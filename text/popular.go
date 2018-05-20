package text

import (
	"sort"
	"strings"

	"github.com/jfk9w-go/dvach"
)

func FormatPopular(list []*dvach.Thread, limit int) []string {
	threads := enrichThreads(list)
	sort.Sort(PopularThreads(threads))
	threads = threads[:limit]

	sb := &strings.Builder{}
	chunks := make([]string, 0)
	for i, thread := range threads {
		if i%10 == 0 {
			if i > 0 {
				chunks = append(chunks, sb.String())
				sb.Reset()
			}
		} else {
			sb.WriteString("\n---\n\n")
		}

		preview := FormatThread(thread)
		sb.WriteString(preview)
	}

	if limit%10 != 0 {
		chunks = append(chunks, sb.String())
	}

	if len(chunks) > 0 {
		last := chunks[len(chunks)-1]
		chunks[len(chunks)-1] = last + "\n\n---\n<b>FIN</b>"
	}

	return chunks
}

type PopularThreads []Thread

func (list PopularThreads) Len() int {
	return len(list)
}

func (list PopularThreads) Less(i, j int) bool {
	return list[i].PostsPerHour > list[j].PostsPerHour ||
		list[i].PostsPerHour == list[j].PostsPerHour && list[i].Num > list[j].Num
}

func (list PopularThreads) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
