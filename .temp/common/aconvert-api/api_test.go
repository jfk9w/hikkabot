package aconvert

import (
	"testing"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

func TestAPI_Convert(t *testing.T) {
	api := NewAPI(&httpx.Config{
		Transport: &httpx.TransportConfig{
			Log: "httpx",
		},
	})

	resp, err := api.Convert("3", &httpx.File{Path: "testdata/15301775501121.webm"})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
