package main

import (
	"log"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/api/reddit"
)

func main() {
	config := new(struct {
		Reddit reddit.Config
	})
	err := flu.Read(flu.File("bin/config.json"), flu.JSON(config))
	if err != nil {
		panic(err)
	}
	c := reddit.NewClient(nil, config.Reddit)
	listing, err := c.GetListing("me_irl", "top", 100)
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", listing)
}
