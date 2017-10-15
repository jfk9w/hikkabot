package dvach

import (
	"testing"
	"time"
)

const testThreadLink = "https://2ch.hk/abu/res/42375.html"

func TestGetThreadFeed(t *testing.T) {
	api := NewAPI(nil, APIConfig{
		ThreadFeedTimeout: 2 * time.Second,
	})

	feed, err := api.GetThreadFeed(testThreadLink, 0)
	if err != nil {
		t.Fatal(err)
	}

	feed1, err := api.GetThreadFeed(testThreadLink, 0)
	if err.Error() != ThreadFeedAlreadyRegistered && feed1 != feed {
		t.Fail()
	}

	if err = feed.Start(); err != nil {
		t.Fatal(err)
	}

	if err = feed.Start(); err.Error() != ThreadFeedAlreadyStarted {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		select {
		case err := <-feed.Err:
			t.Fatal(err)

		case post := <-feed.C:
			t.Log(post)
		}
	}

	if err = feed.Stop(); err != nil {
		t.Fatal(err)
	}

	if err = feed.Stop(); err.Error() != ThreadFeedNotRunning {
		t.Fail()
	}
}
