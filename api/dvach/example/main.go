package main

import (
	"log"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w/hikkabot/api/dvach"
)

func main() {
	config := new(struct {
		Dvach struct {
			Usercode string `json:"usercode"`
		} `json:"dvach"`
	})
	err := flu.Read(flu.File("bin/config.json"), flu.JSON(config))
	if err != nil {
		panic(err)
	}
	c := dvach.NewClient(nil, config.Dvach.Usercode)
	catalog, err := c.GetCatalog("e")
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", catalog.Threads)
	_, err = c.GetThread("tw", 1, 1)
	if err == nil {
		panic("err must not be nil")
	}
	post, err := c.GetPost("e", catalog.Threads[0].Num)
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", post)
	posts, err := c.GetThread("e", catalog.Threads[0].Num, 0)
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", posts)
	board, err := c.GetBoard("b")
	if err != nil {
		panic(err)
	}
	log.Printf("Received %+v", board)
}
