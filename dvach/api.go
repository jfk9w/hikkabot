package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
)

// Endpoint of `2ch.hk`
const Endpoint = "https://2ch.hk"

var (

	// ErrInvalidThreadLink is returned when thread link is malformed
	ErrInvalidThreadLink = errors.New("invalid thread link")
)

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

func (svc *API) GetThread(board string, threadID string, offset int) ([]Post, error) {
	if offset <= 0 {
		offset, _ = strconv.Atoi(threadID)
	}

	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_thread&board=%s&thread=%s&num=%d",
		Endpoint, board, threadID, offset)

	posts := make([]Post, 0)
	if err := httpGetJSON(svc.client, endpoint, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (svc *API) GetPost(board string, num string) ([]Post, error) {
	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_post&board=%s&post=%s",
		Endpoint, board, num)

	posts := make([]Post, 0)
	if err := httpGetJSON(svc.client, endpoint, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

var threadLinkRegexp = regexp.MustCompile(`((http|https):\/\/){0,1}2ch\.hk\/([a-z]+)\/res\/([0-9]+)\.html`)

// FormatThreadURL composes thread URL from board code and thread ID
func FormatThreadURL(board string, threadID string) string {
	return fmt.Sprintf("%s/%s/res/%s.html", Endpoint, board, threadID)
}

// ParseThreadURL extracts board code and thread ID from thread URL
func ParseThreadURL(url string) (string, string, error) {
	groups := threadLinkRegexp.FindSubmatch([]byte(url))
	if len(groups) == 5 {
		return string(groups[3]), string(groups[4]), nil
	}

	return "", "", ErrInvalidThreadLink
}
