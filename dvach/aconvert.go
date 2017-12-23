package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/phemmer/sawmill"

	"github.com/jfk9w/hikkabot/util"
)

const (
	aconvertTimeout    = 500 * time.Millisecond
	maxAconvertRetries = 3
)

type webmRequest struct {
	url     string
	mp4     chan string
	retries int
}

type WebMResult struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

type WebmCache struct {
	client *http.Client
	webms  map[string]chan string
	queue  chan webmRequest
	retry  chan webmRequest
	mu     *sync.Mutex
	halt   util.Hook
	done   util.Hook
}

func newWebmCache(client *http.Client) *WebmCache {
	return &WebmCache{
		client: client,
		webms:  make(map[string]chan string),
		queue:  make(chan webmRequest, 10000),
		retry:  make(chan webmRequest, 10000),
		mu:     new(sync.Mutex),
		halt:   util.NewHook(),
		done:   util.NewHook(),
	}
}

func (svc *WebmCache) Start() {
	sawmill.Info("webm cache started")
	ticker := time.NewTicker(aconvertTimeout)
	go func() {
		defer func() {
			svc.done.Send()
			ticker.Stop()
		}()

		for {
			select {
			case <-ticker.C:
				select {
				case req := <-svc.retry:
					svc.makeRequest(req)

				case req := <-svc.queue:
					svc.makeRequest(req)

				default:
				}

			case <-svc.halt:
				return
			}
		}
	}()
}

func (svc *WebmCache) Preload(webms []string) []chan string {
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

		svc.queue <- webmRequest{
			url: webm,
			mp4: mp4,
		}
	}

	return output
}

func (svc *WebmCache) Get(webm string) string {
	mp4 := svc.Preload([]string{webm})[0]
	link := <-mp4
	mp4 <- link
	return link
}

func (svc *WebmCache) makeRequest(req webmRequest) {
	webm := req.url
	mp4 := req.mp4
	retries := req.retries + 1

	var retry func(error)
	if retries > maxAconvertRetries {
		retry = func(err error) {
			sawmill.Error("webm conversion error", sawmill.Fields{
				"url":   webm,
				"error": err,
			})

			mp4 <- webm
		}
	} else {
		retry = func(err error) {
			sawmill.Warning("webm conversion", sawmill.Fields{
				"url":   webm,
				"error": err,
			})

			svc.retry <- webmRequest{
				url:     webm,
				mp4:     mp4,
				retries: retries,
			}
		}
	}

	go func() {
		sawmill.Debug("webm conversion started", sawmill.Fields{
			"url": webm,
		})

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
			retry(err)
			return
		}

		result := WebMResult{}
		err = parseResponseJSON(resp, &result)
		if err != nil {
			retry(err)
			return
		}

		if result.State != "SUCCESS" {
			err = errors.New(result.State)
			retry(err)
			return
		}

		mp4url := fmt.Sprintf("https://s%s.aconvert.com/convert/p3r68-cdx67/%s",
			result.Server, result.Filename)

		mp4 <- mp4url

		sawmill.Debug("webm conversion finished", sawmill.Fields{
			"url": webm,
			"mp4": mp4url,
		})
	}()
}

func (svc *WebmCache) Dump() map[string]string {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	webms := make(map[string]string, len(svc.webms))
	for webm, mp4 := range svc.webms {
		link := <-mp4
		webms[webm] = link
	}

	svc.halt.Send()
	svc.done.Wait()

	sawmill.Info("webm cache dumped")
	return webms
}
