package dvach

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxWorkers         = 7
	maxAconvertRetries = 3
)

type WebmResult struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

type webmRequest struct {
	url string
	out chan string
}

type WebmCache struct {
	client *http.Client
	webms  map[string]chan string
	queue  chan webmRequest
	mu     *sync.Mutex
	halt   util.Hook
	done   util.Hook
}

func newWebmCache(client *http.Client) *WebmCache {
	return &WebmCache{
		client: client,
		webms:  make(map[string]chan string),
		queue:  make(chan webmRequest, 10000),
		mu:     new(sync.Mutex),
		halt:   util.NewHook(),
		done:   util.NewHook(),
	}
}

func (svc *WebmCache) Start() {
	for i := 0; i <= maxWorkers; i++ {
		go func(server int) {
			sawmill.Info("webm worker started", sawmill.Fields{
				"server": server,
			})

			defer func() {
				sawmill.Info("webm worker stopped", sawmill.Fields{
					"server": server,
				})

				svc.halt.Send()
				svc.done.Send()
			}()

			for {
				select {
				case <-svc.halt:
					return

				case req := <-svc.queue:
					svc.process(server, req)
				}
			}
		}(3 + i*2)
	}
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
			out: mp4,
		}
	}

	return output
}

func (svc *WebmCache) Get(webm string) string {
	mp4 := svc.Preload([]string{webm})[0]
	link := <-mp4
	sawmill.Debug("webm conversion cache", sawmill.Fields{
		"url": webm,
		"mp4": link,
	})

	mp4 <- link
	return link
}

func (svc *WebmCache) process(server int, req webmRequest) {
	webm, mp4 := req.url, req.out
	var last error
	onError := func(err error) {
		mp4 <- webm
		sawmill.Error("webm conversion error", sawmill.Fields{
			"url":    webm,
			"error":  err,
			"server": server,
		})
	}

	onRetry := func(err error) {
		last = err
		sawmill.Warning("webm conversion", sawmill.Fields{
			"url":    webm,
			"error":  err,
			"server": server,
		})
	}

	onSuccess := func(link string, time float64) {
		mp4 <- link
		sawmill.Debug("webm conversion finished", sawmill.Fields{
			"url":    webm,
			"mp4":    link,
			"time":   time,
			"server": server,
		})
	}

	sawmill.Debug("webm conversion started", sawmill.Fields{
		"url":    webm,
		"server": server,
	})

	for i := 0; i < maxAconvertRetries; i++ {
		select {
		case <-svc.halt:
			sawmill.Debug("webm conversion interrupted", sawmill.Fields{
				"url":    webm,
				"server": server,
			})

			svc.halt.Send()
			return

		default:
		}

		start := time.Now()
		resp, err := svc.client.PostForm(
			fmt.Sprintf(
				"https://s%d.aconvert.com/convert/convert-batch.php",
				server),
			url.Values{
				"file":            {webm},
				"targetformat":    {"mp4"},
				"videooptiontype": {"0"},
				//				"videosizetype":     {"640x480"},
				//				"customvideowidth":  {},
				//				"customvideoheight": {},
				//				"videobitratetype":  {"512k"},
				//				"custombitrate":     {},
				//				"frameratetype":     {"23.976"},
				//				"customframerate":   {},
				//				"videoaspect":       {"0"},
				"code":         {"81000"},
				"filelocation": {"online"},
			},
		)

		if err != nil {
			onRetry(err)
			continue
		}

		result := WebmResult{}
		err = parseResponseJSON(resp, &result)
		if err != nil {
			onRetry(err)
			continue
		}

		if result.State != "SUCCESS" {
			err = errors.New(result.State)
			onRetry(err)
			continue
		}

		link := fmt.Sprintf("https://s%s.aconvert.com/convert/p3r68-cdx67/%s",
			result.Server, result.Filename)

		onSuccess(link, time.Now().Sub(start).Seconds())
		return
	}

	onError(last)
}

func (svc *WebmCache) Dump() map[string]string {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.halt.Send()
	for i := 0; i <= maxWorkers; i++ {
		svc.done.Wait()
	}

	webms := make(map[string]string, len(svc.webms))
	for webm, mp4 := range svc.webms {
		select {
		case link := <-mp4:
			webms[webm] = link

		default:
		}
	}

	sawmill.Info("webm cache dumped")
	return webms
}
