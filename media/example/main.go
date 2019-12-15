package main

import (
	"log"
	"os"

	"github.com/jfk9w-go/flu"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w/hikkabot/media"
)

func main() {
	config := media.Config{
		Workers: 1,
		TempDir: "/tmp/hikkabot_media",
		Aconvert: aconvert.Config{
			TestFile:   "mediaBatch/example/testdata/test.webm",
			TestFormat: "webm",
		},
	}
	manager := media.NewManager(config, nil)
	defer manager.Shutdown()
	mediaBatch := media.NewBatch(mockMediaLoder{config.Aconvert.TestFile})
	manager.Download(mediaBatch)
	file, mediaType, err := (&mediaBatch[0]).WaitForResult()
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(file.Path())
	log.Printf("Downloaded resource of type %v", mediaType)
}

type mockMediaLoder struct {
	sourcePath string
}

func (l mockMediaLoder) LoadMedia(resource flu.ResourceWriter) (media.Type, error) {
	os.Link(l.sourcePath, resource.(flu.File).Path())
	return media.WebM, nil
}
