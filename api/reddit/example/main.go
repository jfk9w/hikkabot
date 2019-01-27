package main

import (
	"log"

	"github.com/jfk9w-go/lego"
	"github.com/jfk9w/hikkabot/api/reddit"
)

func main() {
	configPath := "../../../config.json"
	config := new(struct {
		Reddit *reddit.Config `json:"reddit"`
	})

	lego.Check(lego.ReadJSONFromFile(configPath, config))

	c := reddit.NewClient(nil, config.Reddit)
	listing, err := c.GetListing("me_irl", reddit.TopSort, 100)
	lego.Check(err)

	log.Println("Received", listing[0].Data.Title)
}
