package reddit

import (
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
)

type defaultMediaLoader struct {
	thing  *reddit.Thing
	client *reddit.Client
}

func (l defaultMediaLoader) LoadMedia(resource flu.ResourceWriter) (media.Type, error) {
	err := l.client.Download(l.thing, resource)
	if err != nil {
		return 0, errors.Wrapf(err, "on file download")
	} else {
		var type_ media.Type
		switch l.thing.Data.Extension {
		case "gifv", "gif", "mp4":
			type_ = media.Video
		default:
			type_ = media.Photo
		}
		return type_, nil
	}
}
