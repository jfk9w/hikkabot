package main

import (
	"log"

	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/util"
)

func main() {
	config := new(struct {
		Dvach struct {
			Usercode string `json:"usercode"`
		} `json:"dvach"`
	})

	util.ReadJSON("bin/config.json", config)
	c := dvach.NewClient(nil, config.Dvach.Usercode)

	catalog, err := c.GetCatalog("e")
	util.Check(err)
	log.Printf("Received %+v", catalog.Threads)

	_, err = c.GetThread("tw", 1, 1)
	if err == nil {
		panic("err must not be nil")
	}

	post, err := c.GetPost("e", catalog.Threads[0].Num)
	util.Check(err)
	log.Printf("Received %+v", post)

	posts, err := c.GetThread("e", catalog.Threads[0].Num, 0)
	util.Check(err)
	log.Printf("Received %+v", posts)

	board, err := c.GetBoard("b")
	util.Check(err)
	log.Printf("Received %+v", board)
}
