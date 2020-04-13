package main

import (
	"log"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"

	fluhttp "github.com/jfk9w-go/flu/http"

	_aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/media/descriptor"
)

func main() {
	mediator := &media.Tor{
		SizeBounds: [2]int64{1 << 10, 75 << 20},
		Debug:      true,
		Workers:    1,
		Metrics:    metrics.DummyClient{},
	}

	config := new(struct{ Aconvert *_aconvert.Client })
	if err := flu.DecodeFrom(flu.File("config/config_dev.yml"), flu.YAML{config}); err != nil {
		panic(err)
	}
	mediator.AddConverter(media.NewAconvertConverter(config.Aconvert.Init(), media.NewBufferSpace("")))
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
