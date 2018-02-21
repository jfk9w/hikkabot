package webm

import (
	"fmt"
	"net/http"
	u "net/url"
	"time"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

type response struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

type internalServerError struct {
	underlying string
}

func (e internalServerError) Error() string {
	return fmt.Sprintf("aconvert remote error: %s", e.underlying)
}

type defaultClient http.Client

func Wrap(httpc *http.Client) Client {
	return (*defaultClient)(httpc)
}

func (c *defaultClient) Load(endpoint string, url string) (string, error) {
	r, err := (*http.Client)(c).PostForm(
		endpoint,
		u.Values{
			"file":            {url},
			"targetformat":    {"mp4"},
			"videooptiontype": {"0"},
			//"videosizetype":     {"640x480"},
			//"customvideowidth":  {},
			//"customvideoheight": {},
			//"videobitratetype":  {"512k"},
			//"custombitrate":     {},
			//"frameratetype":     {"23.976"},
			//"customframerate":   {},
			//"videoaspect":       {"0"},
			"code":         {"81000"},
			"filelocation": {"online"},
		})

	if err != nil {
		return "", err
	}

	resp := new(response)
	err = util.ReadResponse(r, resp)
	if err != nil {
		return "", err
	}

	if resp.State != "SUCCESS" {
		return "", internalServerError{resp.State}
	}

	return fmt.Sprintf(
		"https://s%s.aconvert.com/convert/p3r68-cdx67/%s",
		resp.Server, resp.Filename), nil
}

type context struct {
	C       chan Request
	client  Client
	cache   Cache
	retries int
	srv     int
}

func (ctx *context) endpoint() string {
	return fmt.Sprintf(
		"https://s%d.aconvert.com/convert/convert-batch.php",
		ctx.srv)
}

func (ctx *context) log() *log.Entry {
	return log.WithFields(log.Fields{
		"srv": ctx.srv,
	})
}

func worker(ctx *context) util.Handle {
	h := util.NewHandle()
	go func() {
		defer h.Reply()
		for {
			select {
			case <-h.C:
				return

			case req := <-ctx.C:
				if !handleRequest(ctx, h, req) {
					return
				}
			}
		}
	}()

	return h
}

func handleRequest(ctx *context, h util.Handle, req Request) bool {
	l := ctx.log().WithFields(log.Fields{
		"url": req.URL,
	})

	for {
		v := ctx.cache.GetWebm(req.URL)
		if v == "" {
			l.Error("WEBM unable to load webm from cache")
			req.C <- Marked
			return true
		}

		var err error
		switch v {
		case NotFound:
			if ctx.cache.UpdateWebm(req.URL, NotFound, Pending) {
				for i := 0; i < ctx.retries; i++ {
					v, err = ctx.client.Load(ctx.endpoint(), req.URL)
					if err == nil {
						break
					}

					select {
					case <-h.C:
						req.C <- Marked
						return false

					default:
						time.Sleep(3 * time.Second)
					}
				}
			} else {
				continue
			}

			if err != nil {
				v = Marked
			}

			ctx.cache.UpdateWebm(req.URL, Pending, v)
			req.C <- v
			return true

		case Pending:
			select {
			case <-h.C:
				req.C <- Marked
				return false

			default:
				time.Sleep(10 * time.Second)
			}

		default:
			req.C <- v
			return true
		}
	}
}
