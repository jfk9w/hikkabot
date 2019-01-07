package main

import (
	"log"

	"github.com/jfk9w-go/hikkabot/api/reddit"
	"github.com/jfk9w-go/lego"
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
