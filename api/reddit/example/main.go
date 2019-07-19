package main

import (
	"log"

	"github.com/jfk9w/hikkabot/util"

	"github.com/jfk9w/hikkabot/api/reddit"
)

func main() {
	config := new(struct {
		Reddit *reddit.Config `json:"reddit"`
	})

	util.ReadJSON("bin/config.json", config)
	c := reddit.NewClient(nil, config.Reddit)
	defer c.Shutdown()

	listing, err := c.GetListing("me_irl", reddit.Top, 100)
	util.Check(err)

	log.Printf("Received %+v", listing)
}
