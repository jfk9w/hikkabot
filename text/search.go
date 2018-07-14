package text

import (
	"sort"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/mathx"
)

func Search(list []*dvach.Thread, searchText []string, sorted bool, count int) []string {
	threads := searchThreads(list, searchText)
	if sorted {
		sort.Sort(PopularThreads(threads))
	}

	count = mathx.MinInt(count, len(threads))
	threads = threads[:mathx.MinInt(30, len(threads))]

	sb := &strings.Builder{}
	hasText := false
	chunks := make([]string, 0)
	for i, thread := range threads {
		if i%10 == 0 {
			if i > 0 {
				chunks = append(chunks, sb.String())
				sb.Reset()
				hasText = false
			}
		} else {
			sb.WriteString("\n---\n\n")
		}

		preview := FormatThread(thread)
		sb.WriteString(preview)
		hasText = true
	}

	if hasText {
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
