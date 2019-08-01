package dvach

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/subscription"
)

func createMedia(ctx subscription.Context, file *dvach.File) media.Media {
	return media.Media{
		Href: dvach.Host + file.Path,
		Factory: func(resource flu.FileSystemResource) (media.Type, error) {
			return mediaType(file), ctx.DvachClient.DownloadFile(file, resource)
		},
	}
}

func mediaType(file *dvach.File) media.Type {
	switch file.Type {
	case dvach.WebM:
		return media.WebM
	case dvach.MP4, dvach.GIF:
		return media.Video
	default:
		return media.Photo
	}
}

var replyRegexp = regexp.MustCompile(`<a\s+href=".*?/([a-zA-Z0-9]+)/res/([0-9]+)\.html#([0-9]+)".*?>.*?</a>`)

func comment(comment string) string {
	matches := replyRegexp.FindAllStringSubmatch(comment, -1)
	for _, match := range matches {
		board := match[1]
		num := match[3]
		comment = strings.Replace(comment, match[0], fmt.Sprintf("#%s%s", strings.ToUpper(board), num), -1)
	}

	return comment
}
