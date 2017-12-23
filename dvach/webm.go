package dvach

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

const (
	aconvertThreshold  = 3
	maxAconvertRetries = 3
)

type WebMResult struct {
	//Server   string `json:"server"`
	//Filename string `json:"filename"`
	//State    string `json:"state"`
	Status string `json:"status"`
	ID     string `json:"id""`
}

type WebmCache struct {
	client *http.Client
	webms  map[string]chan string
	sem    *semaphore.Weighted
	mu     *sync.Mutex
	halt   util.Hook
	done   util.Hook
}

func newWebmCache(client *http.Client) *WebmCache {
	return &WebmCache{
		client: client,
		webms:  make(map[string]chan string),
		sem:    semaphore.NewWeighted(aconvertThreshold),
		mu:     new(sync.Mutex),
		halt:   util.NewHook(),
		done:   util.NewHook(),
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

		go svc.convert(webm, mp4)
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

func (svc *WebmCache) convert(webm string, mp4 chan string) {
	svc.sem.Acquire(context.TODO(), 1)
	defer svc.sem.Release(1)

	var lastErr error

	onRetry := func(err error) {
		lastErr = err
		sawmill.Warning("webm conversion", sawmill.Fields{
			"url":   webm,
			"error": err,
		})

		time.Sleep(1 * time.Second)
	}

	onError := func(err error) {
		mp4 <- webm
		sawmill.Error("webm conversion error", sawmill.Fields{
			"url":   webm,
			"error": err,
		})
	}

	onSuccess := func(link string, time int64) {
		mp4 <- link
		sawmill.Debug("webm conversion finished", sawmill.Fields{
			"url":  webm,
			"mp4":  link,
			"time": time,
		})
	}

	sawmill.Debug("webm conversion started", sawmill.Fields{
		"url": webm,
	})

	for i := 0; i < maxAconvertRetries; i++ {
		select {
		case <-svc.halt:
			svc.halt.Send()
			mp4 <- ""
			return

		default:
		}

		start := time.Now()
		//resp, err := svc.client.PostForm(
		//	"https://s17.aconvert.com/convert/convert-batch.php",
		//	url.Values{
		//		"file":              {webm},
		//		"targetformat":      {"mp4"},
		//		"videooptiontype":   {"1"},
		//		"videosizetype":     {"640x480"},
		//		"customvideowidth":  {},
		//		"customvideoheight": {},
		//		"videobitratetype":  {"384k"},
		//		"custombitrate":     {},
		//		"frameratetype":     {"23.976"},
		//		"customframerate":   {},
		//		"videoaspect":       {"0"},
		//		"code":              {"81000"},
		//		"filelocation":      {"online"},
		//	},
		//)

		resp, err := svc.client.PostForm(
			"https://www.freefileconvert.com/file/url",
			url.Values{
				"_token":        {"wa9T7pVG4stG0iZBdT3M2kE2oDW8vvoa0ji2lIIu"},
				"url":           {webm},
				"output_format": {"mp4"},
				"progress_key":  {"5a3e98e63f329"},
			},
		)

		if err != nil {
			onRetry(err)
			continue
		}

		result := WebMResult{}
		err = parseResponseJSON(resp, &result)
		if err != nil {
			onRetry(err)
			continue
		}

		if result.Status != "success" {
			err = errors.New(result.Status)
			onRetry(err)
			continue
		}

		link := fmt.Sprintf("https://www.freefileconvert.com/file/%s/download", result.ID)

		onSuccess(link, time.Now().Sub(start).Nanoseconds())
		return
	}

	onError(lastErr)
}

func (svc *WebmCache) Dump() map[string]string {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	webms := make(map[string]string, len(svc.webms))
	for webm, mp4 := range svc.webms {
		link := <-mp4
		if len(link) > 0 {
			webms[webm] = link
		}
	}

	svc.halt.Send()

	sawmill.Info("webm cache dumped")
	return webms
}
