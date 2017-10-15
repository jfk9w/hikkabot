package telegram

import (
	"net/url"
	"time"
)

type Request interface {
	Method() string
	Parameters() url.Values
}

type GetUpdatesRequest struct {
	offset         int
	limit          int
	timeout        time.Duration
	allowedUpdates []string
}

func (r GetUpdatesRequest) Method() string {
	return "/getUpdates"
}

func (r GetUpdatesRequest) Parameters() url.Values {
	p := url.Values{}

	if r.offset > 0 {
		p.Set("offset", string(r.offset))
	}

	if r.limit > 0 {
		p.Set("limit", string(r.limit))
	}

	if r.timeout > 0 {
		p.Set("timeout", string(r.timeout))
	}

	for _, au := range r.allowedUpdates {
		p.Add("allowed_updates", au)
	}

	return p
}
