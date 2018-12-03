package content

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/mathx"
	"github.com/jfk9w-go/gox/utf8x"
	. "github.com/jfk9w-go/telegram/html"
)

var (
	dvachLinkHrefRegexp  = regexp.MustCompile(`/([a-zA-Z+])/*`)
	dvachSpanRegexp      = regexp.MustCompile(`<span.*?>`)
	dvachHtmlTagReplacer = strings.NewReplacer(
		"<br>", "\n",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"</span>", "</i>",
	)

	dvachProcessLink = func(token Token, output *Output) {
		if datanum, ok := TokenAttribute(token, "data-num"); ok {
			if href, ok := TokenAttribute(token, "href"); ok {
				var groups = dvachLinkHrefRegexp.FindStringSubmatch(href)
				if len(groups) == 2 {
					var board = dvach.Board(groups[1])
					var ref, err = dvach.ToRef(board, datanum)
					if err == nil {
						output.AppendText(FormatDvachRef(ref) + " ")
						return
					}
				}
			}
		}

		DefaultProcessLink(token, output)
	}

	dvachPostFormat = (&Format{
		ChunkSize:   paddedMaxMessageSize,
		ProcessLink: dvachProcessLink,
	}).SetDefaults()

	dvachPreviewFormat = (&Format{
		ChunkSize:   275,
		MaxChunks:   1,
		ProcessLink: dvachProcessLink,
	}).SetDefaults()

	DvachPreviewsPerMessage = paddedMaxMessageSize / dvachPreviewFormat.ChunkSize
	dvachThreadTagSanitizer = regexp.MustCompile("[^0-9A-Za-zА-Яа-я]+")
)

func FormatDvachRef(ref dvach.Ref) string {
	return fmt.Sprintf("#%s%s", strings.ToUpper(ref.Board), ref.NumString)
}

const maxDvachThreadTagLength = 25

func FormatDvachThreadTag(thread *dvach.Thread) string {
	var tag = utf8x.Head(thread.Subject, maxDvachThreadTagLength, "")
	tag = dvachThreadTagSanitizer.ReplaceAllString(tag, "_")
	var lastUnderscore = utf8x.LastIndexOf(tag, '_', 0, 0)
	if lastUnderscore > 0 && utf8x.Size(tag)-lastUnderscore < 2 {
		tag = utf8x.Slice(tag, 0, lastUnderscore)
	}

	return "#" + tag
}

func dvachItemText(item dvach.Item) string {
	var text = dvachHtmlTagReplacer.Replace(item.Comment)
	text = dvachSpanRegexp.ReplaceAllString(text, "<i>")
	return text
}

func FormatDvachPost(post *dvach.Post, tag string) []string {
	var parts = dvachPostFormat.Format(dvachItemText(post.Item))
	if parts == nil {
		parts = []string{""}
	}

	parts[0] = fmt.Sprintf("%s\n%s\n---\n", tag, FormatDvachRef(post.Ref)) + parts[0]
	if post.Parent == 0 {
		parts[0] = "#THREAD\n" + parts[0]
	}

	return parts
}

const dvachPreviewHeaderFormat = `<b>%s</b>
<a href="%s">[L]</a> / %d / %d / %.2f/hr
---
`

func calculateDvachThreadPace(thread *dvach.Thread) float64 {
	var age = time.Now().Sub(thread.Date).Hours()
	return float64(thread.PostsCount) / age
}

func formatDvachPreview(thread *dvach.Thread) string {
	var parts = dvachPreviewFormat.Format(dvachItemText(thread.Item))
	if parts == nil {
		parts = []string{""}
	}

	var link = dvach.Endpoint + "/" + thread.Board + "/res/" + thread.NumString + ".html"
	parts[0] = fmt.Sprintf(dvachPreviewHeaderFormat,
		thread.DateString, html.EscapeString(link), thread.PostsCount, thread.FilesCount,
		calculateDvachThreadPace(thread)) + parts[0]

	return parts[0]
}

type DvachSortType uint8

const (
	DvachUnsorted DvachSortType = iota
	DvachSortByPace
	DvachSortByNum
)

type DvachThreadsByPace []*dvach.Thread

func (threads DvachThreadsByPace) Len() int {
	return len(threads)
}

func (threads DvachThreadsByPace) Less(i, j int) bool {
	var (
		paceI = calculateDvachThreadPace(threads[i])
		paceJ = calculateDvachThreadPace(threads[j])
	)

	return paceI > paceJ || paceI == paceJ && threads[i].Num > threads[j].Num
}

func (threads DvachThreadsByPace) Swap(i, j int) {
	threads[i], threads[j] = threads[j], threads[i]
}

type DvachThreadsByNum []*dvach.Thread

func (threads DvachThreadsByNum) Len() int {
	return len(threads)
}

func (threads DvachThreadsByNum) Less(i, j int) bool {
	return threads[i].Num < threads[j].Num
}

func (threads DvachThreadsByNum) Swap(i, j int) {
	threads[i], threads[j] = threads[j], threads[i]
}

func SearchDvachCatalog(threads []*dvach.Thread, sortType DvachSortType, query []string, limit int) []*dvach.Thread {
	if query != nil {
		for i := range query {
			query[i] = strings.ToLower(query[i])
		}

		var filtered = make([]*dvach.Thread, 0)

	threads:
		for _, thread := range threads {
			var comment = strings.ToLower(thread.Comment)
			for i := range query {
				if !strings.Contains(comment, query[i]) {
					continue threads
				}
			}

			filtered = append(filtered, thread)
		}

		threads = filtered
	}

	var sorter sort.Interface = nil
	switch sortType {
	case DvachSortByPace:
		sorter = DvachThreadsByPace(threads)
	}

	if sorter != nil {
		sort.Sort(sorter)
	}

	if limit > 0 {
		threads = threads[:mathx.MinInt(limit, len(threads))]
	}

	return threads
}

func FormatDvachCatalog(threads []*dvach.Thread) []string {
	var previews = make([]string, len(threads))
	for i, thread := range threads {
		previews[i] = formatDvachPreview(thread)
	}

	var (
		chunkCount = int(math.Ceil(float64(len(threads)) / float64(DvachPreviewsPerMessage)))
		chunks     = make([]string, chunkCount)
	)

	for i := range chunks {
		chunks[i] = strings.Join(previews[i:mathx.MinInt(i+DvachPreviewsPerMessage, len(previews))], "\n\n---\n\n")
	}

	if chunkCount == 0 {
		chunkCount = 1
		chunks = []string{""}
	}

	chunks[chunkCount-1] = chunks[chunkCount-1] + "\n\n---\n<b>FIN</b>"
	return chunks
}
