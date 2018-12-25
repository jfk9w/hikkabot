package aconvert

import (
	"fmt"
	"time"

	"github.com/jfk9w-go/hikkabot/common/gox/serialx"
	"github.com/jfk9w-go/hikkabot/common/httpx"
	"github.com/pkg/errors"
)

const Endpoint = "https://s%s.aconvert.com"

type Config struct {
	Retries int           `json:"retries"`
	Http    *httpx.Config `json:"http"`
}

type response struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

func (r response) URL() (string, error) {
	if r.State != "SUCCESS" {
		return "", errors.Errorf("invalid state: %s", r.State)
	}

	return fmt.Sprintf(Endpoint, r.Server) + "/convert/p3r68-cdx67/" + r.Filename, nil
}

type balancerResponse struct {
	URL   string
	Error error
}

func (resp *balancerResponse) Status() serialx.Status {
	if resp.Error == nil {
		return serialx.Ok
	}

	return serialx.Failed
}

func (resp *balancerResponse) Delay() time.Duration {
	return 0
}
