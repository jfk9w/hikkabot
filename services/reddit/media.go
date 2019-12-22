package reddit

import (
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/media"
)

type Media struct {
	thing  *reddit.Thing
	client *reddit.Client
}

func (m Media) URL() string {
	return m.thing.Data.URL
}

func (m Media) Download(out flu.Writable) (typ media.Type, err error) {
	err = m.client.Download(m.thing, out)
	if err == nil {
		typ = m.thing.Data.Extension
	}
	return
}
