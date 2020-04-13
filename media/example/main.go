package main

import (
	"log"
	"time"

	fluhttp "github.com/jfk9w-go/flu/http"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/media/descriptor"
)

func main() {
	mediator := &media.Tor{
		SizeBounds: [2]int64{1 << 10, 75 << 20},
		Debug:      true,
		Workers:    1,
	}

	mediator.AddConverter(media.NewAconvertConverter(aconvert.Client{}.Init(), media.NewBufferSpace("")))
	defer mediator.Initialize().Close()

	md, err := descriptor.From(
		fluhttp.NewClient(nil),
		"https://www.youtube.com/watch?v=g-sgw9bPV4A")
	if err != nil {
		panic(err)
	}

	options := media.Options{
		Hashable: true,
		Buffer:   true,
	}

	startTime := time.Now()
	materialized, err := mediator.Submit("", md, options).Materialize()
	log.Printf("Time took: %s", time.Now().Sub(startTime))
	if err != nil {
		log.Fatalf("Error: %v", err)
		return
	}

	log.Printf("Materialized: %v %s", materialized.Metadata, materialized.Type)
}
