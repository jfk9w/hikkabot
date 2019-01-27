package httpx

import (
	"net/http"
)

func ReadToJSON(resp *http.Response, value interface{}) error {
	var (
		json = &JSON{value}
		err  = json.read(resp.Body)
	)

	if err == nil {
		resp.Body.Close()
	}

	return err
}
