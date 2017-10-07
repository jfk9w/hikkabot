package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ThreadFeed struct {
	C   chan Post
	Err chan error

	Board    string
	ThreadID int
	Post     int

	client  *http.Client
	timeout time.Duration
	stop    chan struct{}
	wg      *sync.WaitGroup
	mu      *sync.Mutex
}

func newThreadFeed(client *http.Client, board string, threadId int, post int, timeout time.Duration) *ThreadFeed {
	return &ThreadFeed{
		C:        make(chan Post, 1000),
		Err:      make(chan error, 1),
		Board:    board,
		ThreadID: threadId,
		Post:     post,
		client:   client,
		timeout:  timeout,
		stop:     make(chan struct{}, 1),
		wg:       nil,
		mu:       &sync.Mutex{},
	}
}

func (f *ThreadFeed) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.wg != nil {
		return errors.New(ThreadFeedAlreadyStarted)
	}

	f.wg = &sync.WaitGroup{}
	f.wg.Add(1)

	ticker := time.NewTicker(f.timeout)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-f.stop:
				f.wg.Done()
				return

			case <-ticker.C:
				url := fmt.Sprintf("%s/makaba/mobile.fcgi?task=get_thread&board=%s&thread=%d&num=%d",
					Endpoint, f.Board, f.ThreadID, f.Post)

				posts := make([]Post, 0)
				if err := httpGetJSON(f.client, url, &posts); err != nil {
					f.Err <- err

					f.mu.Lock()
					defer f.mu.Unlock()

					f.wg.Done()
					f.wg = nil
					return
				}

				for _, post := range posts {
					f.C <- post
				}

				if len(posts) > 0 {
					f.Post = posts[len(posts)-1].num() + 1
				}
			}
		}
	}()

	return nil
}

func (f *ThreadFeed) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.wg == nil {
		return errors.New(ThreadFeedNotRunning)
	}

	f.stop <- unit
	f.wg.Wait()
	f.wg = nil

	return nil
}
