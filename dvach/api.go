package dvach

import (
	"regexp"
	"net/http"
	"errors"
	"strconv"
	"sync"
	"fmt"
	"time"
)

const Endpoint = "https://2ch.hk/"

const ThreadFeedAlreadyRegistered = "thread feed already registered"

const threadFeedTimeout = 5 * time.Second

type API struct {
	client  *http.Client
	cfg     APIConfig
	threads map[string]*ThreadFeed
	mu      *sync.Mutex
}

type APIConfig struct {
	ThreadFeedTimeout time.Duration
}

func NewAPI(cfg APIConfig) *API {
	return NewAPIWithClient(&http.Client{}, cfg)
}

func NewAPIWithClient(client *http.Client, cfg APIConfig) *API {
	return &API{
		client:  client,
		cfg:     cfg,
		threads: make(map[string]*ThreadFeed),
		mu:      &sync.Mutex{},
	}
}

func (svc *API) GetThreadFeed(url string, post int) (*ThreadFeed, error) {
	board, threadId, err := parseThreadURL(url)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/%d", board, threadId)

	svc.mu.Lock()
	defer svc.mu.Unlock()

	if f, ok := svc.threads[key]; ok {
		return f, errors.New(ThreadFeedAlreadyRegistered)
	}

	if post <= 0 {
		post = threadId
	}

	f := newThreadFeed(svc.client, board, threadId, post, svc.cfg.ThreadFeedTimeout)
	svc.threads[key] = f

	return f, nil
}

var threadlinkre, _ = regexp.Compile(`((http|https):\/\/){0,1}2ch\.hk\/([a-z]+)\/res\/([0-9]+)\.html`)

func parseThreadURL(url string) (string, int, error) {
	groups := threadlinkre.FindSubmatch([]byte(url))
	if len(groups) == 5 {
		board := string(groups[3])
		threadId, err := strconv.Atoi(string(groups[4]))
		if err != nil {
			return "", -1, errors.New("invalid thread ID: " + err.Error())
		}

		return board, threadId, nil
	}

	return "", -1, errors.New("invalid thread link")
}
