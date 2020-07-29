package descriptor

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jfk9w-go/flu"
	"github.com/stretchr/testify/assert"
)

func TestRedgifsHTMLHandler_Handle(t *testing.T) {
	resp := new(http.Response)
	file, err := flu.File("testdata/redgifs.html").Reader()
	if err != nil {
		t.Fatal(err)
	}
	resp.Body = ioutil.NopCloser(file)
	h := new(redgifsHTMLHandler)
	err = h.Handle(resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "https://thcf6.redgifs.com/DearestContentHydra.mp4", h.URL)
}
