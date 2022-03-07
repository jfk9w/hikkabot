package viddit

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var URL = "https://viddit.red"

type Client struct {
	*http.Client
	mu     flu.RWMutex
	wg     flu.WaitGroup
	cancel func()
}

func (c *Client) RefreshInBackground(ctx context.Context, every time.Duration) error {
	if c.cancel != nil {
		return nil
	}

	if err := c.refresh(ctx); err != nil {
		return err
	}

	c.cancel = c.wg.Go(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for err := c.refresh(ctx); err != nil; err = c.refresh(ctx) {
					if flu.IsContextRelated(err) {
						return
					}

					logrus.Warnf("refresh viddit cookie: %s", err)
				}
			}
		}
	})

	return nil
}

func (c *Client) refresh(ctx context.Context) error {
	defer c.mu.Lock().Unlock()
	jar, err := cookiejar.New(nil)
	if err != nil {
		return errors.Wrap(err, "create new cookie jar")
	}

	c.Jar = jar
	return httpf.GET(URL).
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		Error()
}

func (c *Client) ResolveURL(ctx context.Context, url string) (string, error) {
	defer c.mu.RLock().Unlock()
	h := new(responseHandler)
	return h.url, httpf.GET(URL).
		Query("url", url).
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		Handle(h).
		Error()
}

func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
		c.wg.Wait()
	}

	return nil
}
