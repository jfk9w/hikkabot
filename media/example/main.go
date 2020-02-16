package main

import (
	"log"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/media/descriptor"
	"github.com/rivo/duplo"
)

func main() {
	mediator := &media.Tor{
		SizeBounds: [2]int64{1 << 10, 75 << 20},
		Debug:      true,
		ImgHashes:  duplo.New(),
		Workers:    1,
	}

	mediator.AddConverter(media.NewAconvertConverter(new(aconvert.Config)))
	defer mediator.Initialize().Close()

	md, err := descriptor.From(
		flu.NewClient(nil),
		"https://www.youtube.com/watch?v=g-sgw9bPV4A")
	if err != nil {
		panic(err)
	}

	options := media.Options{
		Hashable: true,
		Buffer:   true,
		//OCR: &media.OCR{
		//	Languages: []string{"rus"},
		//	Regex:     regexp.MustCompile(`д\s?е\s?в\s?с\s?т\s?в\s?е\s?н\s?н\s?и\s?к`),
		//},
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
