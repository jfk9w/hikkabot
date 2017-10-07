package parser

import (
	"testing"
	"time"
	"github.com/jfk9w/tele2ch/dvach"
)

const testThreadLink = "https://2ch.hk/abu/res/42375.html"

func Test_Parse(t *testing.T) {
	api := dvach.NewAPI(dvach.APIConfig{
		ThreadFeedTimeout: 2 * time.Second,
	})

	feed, _ := api.GetThreadFeed(testThreadLink, 0)
	feed.Start()

	for i := 0; i < 5; i++ {
		select {
		case err := <-feed.Err:
			t.Fatal(err)

		case post := <-feed.C:
			t.Log(Parse(post))
		}
	}

	feed.Stop()
}