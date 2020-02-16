package main

import (
	"fmt"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
	"github.com/rivo/duplo"
)

func main() {
	mediator := &media.Tor{
		SizeBounds: [2]int64{1 << 10, 75 << 20},
		Debug:      true,
		ImgHashes:  duplo.New(),
	}

	mediator.AddConverter(media.NewAconvertConverter(new(aconvert.Config)))
	mediator.Initialize()

	descriptor := media.URLDescriptor{
		Client: flu.NewClient(nil),
		URL:    "https://2ch.hk/b/src/213662839/15817545420432.jpg",
	}

	options := media.Options{
		Hashable: true,
		//OCR: &media.OCR{
		//	Languages: []string{"rus"},
		//	Regex:     regexp.MustCompile(`д\s?е\s?в\s?с\s?т\s?в\s?е\s?н\s?н\s?и\s?к`),
		//},
	}

	materialized, err := mediator.Materialize(&descriptor, options)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Materialized:\nmetadata = %v\nmedia type = %s\n",
		materialized.Metadata, materialized.Type)

	//descriptor.URL = "https://2ch.hk/b/src/213696231/15817911669640.png"
	materialized, err = mediator.Materialize(&descriptor, options)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Materialized:\nmetadata = %v\nmedia type = %s\n",
		materialized.Metadata, materialized.Type)
}
