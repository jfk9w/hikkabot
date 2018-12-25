package transport

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/jfk9w-go/hikkabot/common/logx"
)

type Logx struct {
	http.RoundTripper
	logx.Ptr
}

func (t *Logx) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		id        = t.onRequest(req)
		start     = time.Now()
		resp, err = t.RoundTripper.RoundTrip(req)
		duration  = since(start)
	)

	if err != nil {
		t.onError(id, req, duration, err)
	} else {
		t.onResponse(id, resp, duration)
	}

	return resp, err
}

const _64K = 64 * (2 << 10)

func (t *Logx) onRequest(req *http.Request) string {
	var (
		id   = id()
		data []byte
		body io.ReadCloser
	)

	data, req.Body = scan(req.Body)
	body = ioutil.NopCloser(bytes.NewReader(data))
	if req.MultipartForm != nil {
		req.ParseMultipartForm(_64K)
		req.Body = body
		var files = make(map[string][]string)
		for k, headers := range req.MultipartForm.File {
			var names = make([]string, len(headers))
			for j := range headers {
				names[j] = headers[j].Filename
			}

			files[k] = names
		}

		t.Debugf("%s %s > %s%s%s", id, req.Method, req.URL,
			kvs2string("Headers", req.Header, " "),
			kvs2string("Form", req.MultipartForm.Value, " "),
			kvs2string("Files", files, " "))
	} else {
		req.ParseForm()
		req.Body = body
		t.Debugf("%s %s > %s%s%s", id, req.Method, req.URL,
			kvs2string("Headers", req.Header, " "),
			kvs2string("Form", req.Form, " "))
	}

	return id
}

func (t *Logx) onResponse(id string, resp *http.Response, millis string) {
	var (
		contentType = resp.Header.Get("Content-Type")
		body        string
	)

	if !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(contentType, "video/") {
		var data []byte
		data, resp.Body = scan(resp.Body)
		body = "\nBody: " + string(data)
	}

	t.Debugf("%s %s < %s %d\nDuration: %s ms.%s%s",
		id, resp.Request.Method, resp.Request.URL, resp.StatusCode, millis,
		kvs2string("Headers", resp.Header, " "),
		body)
}

func (t *Logx) onError(id string, req *http.Request, millis string, err error) {
	t.Warnf("%s %s < %s\nDuration: %s ms.\nError: %s", id, req.Method, req.URL, millis, err)
}
