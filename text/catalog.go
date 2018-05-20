package text

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/misc"
)

var dvachLocation *time.Location

func init() {
	loc, err := time.LoadLocation(dvach.LocationName)
	if err != nil {
		panic(err)
	}

	dvachLocation = loc
}

type Thread struct {
	*dvach.Thread
	PostsPerHour float64
}

func FormatCatalog(catalog *dvach.Catalog, limit int) []string {
	now := time.Now()
	threads := make([]Thread, len(catalog.Threads))
	for i, thread := range catalog.Threads {
		date, ok := dvach.ToTime(thread.DateString, dvachLocation)
		if !ok {
			continue
		}

		age := now.Sub(date).Hours()
		threads[i] = Thread{thread, float64(thread.PostsCount) / age}
	}

	sort.Sort(ThreadList(threads))
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

	return chunks
}

//var threadHeaderSanitizer = regexp.MustCompile("<.*?>")

func FormatThread(thread Thread) string {
	chunks := format(thread.Item, 275)
	if len(chunks) == 0 {
		return ""
	}

	//header := threadHeaderSanitizer.ReplaceAllString(thread.Subject, "")
	//header = "<b>" + misc.FirstRunes(header, 70, "...") + "</b>"

	num := fmt.Sprintf("%s / %s", FormatRef(thread.Ref), thread.DateString)
	stats := fmt.Sprintf("%d ps / %.2f ps/h", thread.PostsCount, thread.PostsPerHour)
	content := misc.FirstRunes(chunks[0], 275, "...")

	return fmt.Sprintf("%s\n%s\n---\n%s", num, stats, content)
}

type ThreadList []Thread

func (list ThreadList) Len() int {
	return len(list)
}

func (list ThreadList) Less(i, j int) bool {
	return list[i].PostsPerHour > list[j].PostsPerHour || list[i].Num > list[j].Num
}

func (list ThreadList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
