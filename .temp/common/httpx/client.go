package httpx

import (
	"bytes"
	"mime/multipart"
	"net/http"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/segmentio/ksuid"
)

// T is a wrapper type around http.Client.
type T struct {
	*http.Client

	// TempStorage is a directory for received files.
	TempStorage string
}

// Get issues a GET request.
func (obj *T) Get(url string, input Params, output Output) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	obj.temp(req, output)

	req.URL.RawQuery = input.Encode()
	return obj.exec(req, output)
}

// Post issues a POST request.
func (obj *T) Post(url string, input POST, output Output) error {
	buf := new(bytes.Buffer)
	input.write(buf)

	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return err
	}

	obj.temp(req, output)

	req.Header.Set("Content-Type", input.contentType())
	return obj.exec(req, output)
}

// Multipart issues a POST request and uploads files
// as multipart/form-data.
func (obj *T) Multipart(url string, params Params, input Multipart, output Output) error {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	for key, file := range input {
		part, err := writer.CreateFormFile(key, key)
		if err != nil {
			return err
		}

		err = file.write(part)
		if err != nil {
			return err
		}
	}

	for key, values := range params {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return obj.exec(req, output)
}

func (obj *T) temp(req *http.Request, output Output) {
	if obj.TempStorage != "" {
		if file, ok := output.(*File); ok {
			if file.Path == "" {
				file.Path = fsx.Join(obj.TempStorage, ksuid.New().String())
			}
		}
	}
}

func (obj *T) exec(req *http.Request, output Output) error {
	resp, err := obj.Client.Do(req)
	if err != nil {
		return err
	}

	err = output.read(resp.Body)
	resp.Body.Close()
	return err
}
