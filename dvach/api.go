package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
)

const Endpoint = "https://2ch.hk/"

type API struct {
	client *http.Client
}

func NewAPI(client *http.Client) *API {
	if client == nil {
		client = new(http.Client)
	}

	return &API{
		client: client,
	}
}

func (svc *API) GetThread(url string, offset int) ([]Post, error) {
	board, threadId, err := parseThreadURL(url)
	if err != nil {
		return nil, err
	}

	if offset <= 0 {
		offset = threadId
	}

	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_thread&board=%s&thread=%d&num=%d",
		Endpoint, board, threadId, offset)

	posts := make([]Post, 0)
	if err := httpGetJSON(svc.client, endpoint, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

var threadLinkRegexp = regexp.MustCompile(`((http|https):\/\/){0,1}2ch\.hk\/([a-z]+)\/res\/([0-9]+)\.html`)

func parseThreadURL(url string) (string, int, error) {
	groups := threadLinkRegexp.FindSubmatch([]byte(url))
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
