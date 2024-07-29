package httpf

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
)

type multipartFile struct {
	name  string
	input flu.Input
}

// MultipartForm represents a multipart/form-data request body.
type MultipartForm struct {
	// Form contains form values (except for files).
	Form
	boundary string
	files    map[string]multipartFile
}

// Set sets form values.
func (mf *MultipartForm) Set(key string, values ...string) *MultipartForm {
	if mf.err != nil {
		return mf
	}

	(&mf.Form).Set(key, values...)
	return mf
}

// File adds a file to the form data.
func (mf *MultipartForm) File(fieldName, filename string, input flu.Input) *MultipartForm {
	if mf.err != nil {
		return mf
	}

	if mf.files == nil {
		mf.files = make(map[string]multipartFile)
	}

	if filename == "" {
		filename = fieldName
	}

	mf.files[fieldName] = multipartFile{
		name:  filename,
		input: input,
	}

	return mf
}

func randomBoundary() (string, error) {
	var buf [30]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		return "", errors.Wrap(err, "generate multipart form boundary")
	}

	return fmt.Sprintf("%x", buf[:]), nil
}

func (mf *MultipartForm) EncodeTo(w io.Writer) error {
	if mf.err != nil {
		return mf.err
	}

	mw := multipart.NewWriter(w)
	defer flu.CloseQuietly(mw)

	if err := mw.SetBoundary(mf.boundary); err != nil {
		return errors.Wrap(err, "set multipart form boundary")
	}

	for fieldName, file := range mf.files {
		w, err := mw.CreateFormFile(fieldName, file.name)
		if err != nil {
			return errors.Wrapf(err, "create form file %s (%s)", fieldName, file.name)
		}

		if _, err := flu.Copy(file.input, flu.IO{W: w}); err != nil {
			return errors.Wrapf(err, "copy form file %s (%s", fieldName, file.name)
		}
	}

	if err := writeMultipartValues(mw, mf.values); err != nil {
		return errors.Wrap(err, "write multipart form values")
	}

	return nil
}

func writeMultipartValues(mw *multipart.Writer, uv url.Values) error {
	for k, vs := range uv {
		for _, value := range vs {
			if err := mw.WriteField(k, value); err != nil {
				return errors.Wrapf(err, "write multipart value %s = %v", k, value)
			}
		}
	}
	return nil
}

func (mf *MultipartForm) ContentType() string {
	return "multipart/form-data; boundary=" + mf.boundary
}
