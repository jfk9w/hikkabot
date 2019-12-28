package main

import (
	"log"
	"os"

	"github.com/jfk9w-go/flu"

	aconvert "github.com/jfk9w-go/aconvert-api"

	"github.com/jfk9w/hikkabot/media"
)

var config = media.Config{
	Concurrency: 1,
	Aconvert:    new(aconvert.Config),
}

func main() {
	manager := media.NewManager(config)
	defer manager.Shutdown()
	file := flu.File("media/example/testdata/test.webm")
	media := media.New(file.Path(), "webm", SizeAwareReadable{file})
	manager.Submit(media)
	in, err := media.Ready()
	if err != nil {
		panic(err)
	}
	log.Printf("Processed resource of type %v", in.Type)
}

type SizeAwareReadable struct {
	flu.File
}

func (r SizeAwareReadable) Size() (size int64, err error) {
	stat, err := os.Stat(r.File.Path())
	if err != nil {
		return
	}
	size = stat.Size()
	return
}
