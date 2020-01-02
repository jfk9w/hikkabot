package main

import (
	"log"
	"os"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	. "github.com/jfk9w/hikkabot/mediator"
	mreq "github.com/jfk9w/hikkabot/mediator/request"
)

var config = Config{
	Concurrency: 1,
	Aconvert:    new(aconvert.Config),
}

func main() {
	mediator := New(config)
	defer mediator.Shutdown()
	//url := "mediator/example/testdata/test.webm"
	//request := MediatorRequest{flu.File(url)}
	url := "https://www.youtube.com/watch?v=aIdItDG-FfQ&feature=youtu.be"
	request := &mreq.Youtube{URL: url, MaxSize: MaxSize(telegram.Video)[1]}
	//url := "https://2ch.hk/b/src/210545730/15778681235240.mp4"
	//request := &HTTPRequest{URL: url, Format: "webm"}
	future := mediator.Submit(url, request)
	media, err := future.Result()
	if err != nil {
		panic(err)
	}
	log.Printf("%+v", media)
}

type MediatorRequest struct {
	flu.File
}

func (r MediatorRequest) Metadata() (*Metadata, error) {
	stat, err := os.Stat(r.Path())
	if err != nil {
		return nil, err
	}
	return &Metadata{
		URL:    r.Path(),
		Size:   stat.Size(),
		Format: "webm",
	}, nil
}
