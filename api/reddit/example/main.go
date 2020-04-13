package main

import (
	"log"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w/hikkabot/api/reddit"
)

func main() {
	config := new(struct {
		Reddit reddit.Config
	})
	err := flu.DecodeFrom(flu.File("config/config_dev.yml"), flu.YAML{config})
	if err != nil {
		panic(err)
	}
	c := reddit.NewClient(fluhttp.NewClient(nil), config.Reddit)
	listing, err := c.GetListing("me_irl", "top", 100)
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", listing)
}
