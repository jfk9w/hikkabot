package resolver_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/stretchr/testify/assert"
)

func TestRedGIFs_Handle(t *testing.T) {
	resp := new(http.Response)
	file, err := flu.File("testdata/redgifs.html").Reader()
	if err != nil {
		t.Fatal(err)
	}

	resp.Body = ioutil.NopCloser(file)
	r := new(resolver.RedGIFs)
	err = r.Handle(resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "https://thcf6.redgifs.com/DearestContentHydra.mp4", r.URL)
}
