package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/phemmer/sawmill"
)

// Endpoint of `2ch.hk`
const (
	Endpoint       = "https://2ch.hk"
	TypeWebM       = 6
	BatchPostCount = 20
)

var (

	// ErrInvalidThreadLink is returned when thread link is malformed
	ErrInvalidThreadLink = errors.New("invalid thread link")
)

type API struct {
	client *http.Client
	webm   *WebmCache
}

func NewAPI(client *http.Client) *API {
	if client == nil {
		client = new(http.Client)
	}

	return &API{
		client: client,
		webm:   newWebmCache(client),
	}
}

func (svc *API) Start() {
	svc.webm.Start()
}

func (svc *API) Stop() {
	svc.webm.Dump()
}

func (svc *API) GetThread(board string, threadID string, offset int) ([]Post, error) {
	if offset <= 0 {
		offset, _ = strconv.Atoi(threadID)
	}

	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_thread&board=%s&thread=%s&num=%d",
		Endpoint, board, threadID, offset)

	sawmill.Debug("get thread", sawmill.Fields{
		"url": endpoint,
	})

	posts := make([]Post, 0)
	resp, err := svc.client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	if err := parseResponseJSON(resp, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (svc *API) GetPost(board string, num string) ([]Post, error) {
	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_post&board=%s&post=%s",
		Endpoint, board, num)

	sawmill.Debug("get post", sawmill.Fields{
		"url": endpoint,
	})

	posts := make([]Post, 0)
	resp, err := svc.client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	if err := parseResponseJSON(resp, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (svc *API) GetFiles(post Post) map[string]string {
	webms := make(map[string]string)
	for _, file := range post.Files {
		if file.Type == TypeWebM {
			webms[file.URL()] = svc.webm.Get(file.URL())
		}
	}

	return webms
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
