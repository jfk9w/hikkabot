package main

import (
	"fmt"
	"os"

	"github.com/jfk9w-go/flu"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w/hikkabot/media"
)

func main() {
	config := media.Config{
		Workers: 1,
		TempDir: "/tmp/hikkabot_media",
	}
	aconvertConfig := aconvert.Config{
		TestFile:   "app/media/example/testdata/test.webm",
		TestFormat: "webm",
	}

	aconvertClient := aconvert.NewClient(nil, &aconvertConfig)
	mediaManager := media.NewManager(config, aconvertClient)
	defer mediaManager.Shutdown()

	me := []media.Media{{
		Href: "http://test",
		Factory: func(resource flu.FileSystemResource) (media.Type, error) {
			os.Link(aconvertConfig.TestFile, resource.Path())
			return media.WebM, nil
		},
	}}

	mediaManager.Download(me)
	r, t, err := me[0].Get()
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(r.Path())
	fmt.Printf("Downloaded resource of type %v", t)
}
