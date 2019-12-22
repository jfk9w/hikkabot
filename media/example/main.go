package main

import (
	"errors"
	"log"

	"github.com/jfk9w-go/flu"

	aconvert "github.com/jfk9w-go/aconvert-api"

	"github.com/jfk9w/hikkabot/media"
)

var config = media.Config{
	Concurrency: 1,
	Dir:         "/tmp/hikkabot_media",
	Aconvert:    new(aconvert.Config),
}

func main() {
	manager := media.NewManager(config)
	defer manager.Shutdown()
	media := Media{"media/example/testdata/test.webm"}
	res, typ, err := manager.Download(media)[0].Wait()
	if err != nil {
		panic(err)
	}
	defer res.Cleanup()
	log.Printf("Download resource of type %v", typ)
}

type Media struct {
	Source flu.File
}

func (m Media) URL() string {
	return m.Source.Path()
}

func (m Media) Download(out flu.Writable) (typ media.Type, err error) {
	typ = "webm"
	file, ok := out.(media.File)
	if !ok {
		err = errors.New("out should be a media.File")
		return
	}
	err = flu.Copy(m.Source, file)
	return
}
