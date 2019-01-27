package aconvert

import (
	"fmt"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

type API struct {
	httpx.HTTP
}

func NewAPI(config *httpx.Config) *API {
	return &API{httpx.Configure(config)}
}

func (api API) Convert(server string, input interface{}) (string, error) {
	var (
		url    = fmt.Sprintf(Endpoint, server) + "/convert/convert-batch.php"
		resp   = new(response)
		output = &httpx.JSON{Value: resp}
		err    error
	)

	params := httpx.Params{
		"targetformat":    {"mp4"},
		"videooptionsize": {"0"},
		"code":            {"81000"},
	}

	switch input := input.(type) {
	case string:
		err = api.Post(url, params.Set(
			"filelocation", "online").Set(
			"file", input), output)

	case *httpx.File:
		err = api.Multipart(url, params.Set(
			"filelocation", "local"), httpx.Multipart{
			"file": input}, output)

	default:
		panic(fmt.Sprintf("invalid input type: %T", input))
	}

	if err != nil {
		return "", err
	}

	url, err = resp.URL()
	return url, err
}
