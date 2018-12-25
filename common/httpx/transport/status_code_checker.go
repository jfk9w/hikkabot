package transport

import (
	"net/http"

	"github.com/pkg/errors"
)

type StatusCodeChecker struct {
	http.RoundTripper
	Codes []int
}

func (t *StatusCodeChecker) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for _, code := range t.Codes {
		if code == resp.StatusCode {
			return resp, nil
		}
	}

	return nil, errors.Errorf("invalid status code: %d", resp.StatusCode)
}
