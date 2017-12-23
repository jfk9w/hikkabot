package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/phemmer/sawmill"
)

// Endpoint of `2ch.hk`
const (
	Endpoint       = "https://2ch.hk"
	TypeWebM       = 6
	maxWebMRetries = 3
)

var (

	// ErrInvalidThreadLink is returned when thread link is malformed
	ErrInvalidThreadLink = errors.New("invalid thread link")
)

type API struct {
	client *http.Client
	webms  map[string]chan string
	mu     *sync.Mutex
	wg     *sync.WaitGroup
}

type WebMResult struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

func NewAPI(client *http.Client) *API {
	if client == nil {
		client = new(http.Client)
	}

	return &API{
		client: client,
		webms:  make(map[string]chan string),
		mu:     new(sync.Mutex),
		wg:     new(sync.WaitGroup),
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
	resp, err := svc.client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	if err := parseResponseJSON(resp, &posts); err != nil {
		return nil, err
	}

	webms := make([]string, 0)
	for _, post := range posts {
		for _, file := range post.Files {
			if file.Type == TypeWebM {
				webms = append(webms, file.URL())
			}
		}
	}

	go svc.preloadWebMs(webms)
	for _, post := range posts {
		for _, file := range post.Files {
			if file.Type == TypeWebM {
				file.url = svc.convert(file.URL())
			}
		}
	}

	return posts, nil
}

func (svc *API) GetPost(board string, num string) ([]Post, error) {
	endpoint := fmt.Sprintf(
		"%s/makaba/mobile.fcgi?task=get_post&board=%s&post=%s",
		Endpoint, board, num)

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

func (svc *API) preloadWebMs(webms []string) []chan string {
	if len(webms) == 0 {
		return nil
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	output := make([]chan string, len(webms))
	for i, webm := range webms {
		if mp4, ok := svc.webms[webm]; ok {
			output[i] = mp4
			continue
		}

		mp4 := make(chan string, 1)
		output[i] = mp4
		svc.webms[webm] = mp4

		go svc.convertWebM(webm, mp4)
	}

	return output
}

func (svc *API) convertWebM(webm string, mp4 chan string) {
	svc.wg.Add(1)
	defer svc.wg.Done()

	sawmill.Debug("webm conversion started", sawmill.Fields{
		"url": webm,
	})

	var lastErr error
	for i := 0; i < maxWebMRetries; i++ {
		resp, err := svc.client.PostForm(
			"https://s17.aconvert.com/convert/convert-batch.php",
			url.Values{
				"file":              {webm},
				"targetformat":      {"mp4"},
				"videooptiontype":   {"0"},
				"videosizetype":     {"0"},
				"customvideowidth":  {},
				"customvideoheight": {},
				"videobitratetype":  {"0"},
				"custombitrate":     {},
				"frameratetype":     {"0"},
				"customframerate":   {},
				"videoaspect":       {"0"},
				"code":              {"81000"},
				"filelocation":      {"online"},
			},
		)

		if err != nil {
			lastErr = err
			sawmill.Warning("webm conversion", sawmill.Fields{
				"url":   webm,
				"error": err,
			})

			continue
		}

		result := WebMResult{}
		err = parseResponseJSON(resp, &result)
		if err != nil {
			lastErr = err
			sawmill.Warning("webm conversion", sawmill.Fields{
				"url":   webm,
				"error": err,
			})

			continue
		}

		if result.State != "SUCCESS" {
			lastErr = errors.New(result.State)
			sawmill.Warning("webm conversion", sawmill.Fields{
				"url":   webm,
				"error": lastErr,
			})

			continue
		}

		mp4url := fmt.Sprintf("https://s%s.aconvert.com/convert/p3r68-cdx67/%s",
			result.Server, result.Filename)

		mp4 <- mp4url

		sawmill.Debug("webm conversion finished", sawmill.Fields{
			"url": webm,
			"mp4": mp4url,
		})

		return
	}

	sawmill.Error("webm conversion error", sawmill.Fields{
		"url":   webm,
		"error": lastErr,
	})

	mp4 <- webm
}

func (svc *API) convert(webm string) string {
	mp4 := svc.preloadWebMs([]string{webm})[0]
	link := <-mp4
	mp4 <- link
	return link
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
