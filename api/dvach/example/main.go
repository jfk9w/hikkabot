package main

import (
	"log"

	"github.com/jfk9w-go/lego"
	"github.com/jfk9w/hikkabot/api/dvach"
)

func main() {
	configPath := "../../../config.json"
	config := new(struct {
		Dvach struct {
			Usercode string `json:"usercode"`
		} `json:"dvach"`
	})

	lego.Check(lego.ReadJSONFromFile(configPath, config))

	c := dvach.NewClient(nil, config.Dvach.Usercode)
	catalog, err := c.GetCatalog("e")
	lego.Check(err)

	log.Println("Received", catalog.Threads[0].Subject)

	_, err = c.GetThread("tw", 1, 1)
	if err == nil {
		panic("err must not be nil")
	}

	log.Println("Received", err)

	board, err := c.GetBoard("b")
	lego.Check(err)

	log.Println("Received", board)
}
