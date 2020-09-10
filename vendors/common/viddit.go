package common

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

var VidditURL = "https://viddit.red"

type Viddit struct {
	*fluhttp.Client
	flu.Mutex
	format.Clock
	ResetInterval time.Duration
	lastResetTime time.Time
}

func (v *Viddit) Get(ctx context.Context, url string) (string, error) {
	defer v.Lock().Unlock()
	now := v.Now()
	if now.Sub(v.lastResetTime) > v.ResetInterval {
		v.Jar = nil
		err := v.WithCookies().GET(VidditURL).
			Context(ctx).
			Execute().
			CheckStatus(http.StatusOK).
			Error
		if err != nil {
			return "", errors.Wrap(err, "refresh cookie")
		}

		v.lastResetTime = now
		log.Print("[viddit] refreshed cookie")
	}

	h := new(vidditResponseHandler)
	err := v.GET(VidditURL).
		QueryParam("url", url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(h).
		Error
	if err != nil {
		return "", errors.Wrap(err, "convert")
	}

	return h.url, nil
}

type vidditResponseHandler struct {
	url string
}

func (h *vidditResponseHandler) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "a" {
			attrs := format.HTMLAttributes(token.Attr)
			if attrs.Get("id") == "dlbutton" {
				h.url = attrs.Get("href")
				return nil
			}
		}
	}

	return errors.New("unable to find url")
}
