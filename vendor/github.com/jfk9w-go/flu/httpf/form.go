package httpf

import (
	"io"
	"net/url"

	"github.com/pkg/errors"

	"github.com/google/go-querystring/query"
)

// Form represents a HTTP form request body.
type Form struct {
	values url.Values
	err    error
}

// FormValue creates a Form based on this value.
// It uses `url` tags for property name resolution.
func FormValue(value interface{}) *Form {
	var (
		values = make(url.Values)
		err    error
	)

	if encoder, ok := value.(query.Encoder); ok {
		err = encoder.EncodeValues("", &values)
	} else {
		values, err = query.Values(value)
	}

	return &Form{values, errors.Wrap(err, "extract form values")}
}

func (f *Form) EncodeTo(writer io.Writer) error {
	if f.err != nil {
		return f.err
	}

	_, err := io.WriteString(writer, f.values.Encode())
	return errors.Wrap(err, "write encoded form values")
}

func (*Form) ContentType() string {
	return "application/x-www-form-urlencoded"
}

// Set sets values to the key.
func (f *Form) Set(key string, values ...string) *Form {
	if f.err != nil {
		return f
	}

	if f.values == nil {
		f.values = make(url.Values)
	}

	f.values.Del(key)
	for _, value := range values {
		f.values.Add(key, value)
	}

	return f
}

// SetAll adds all values from url.Values.
func (f *Form) SetAll(values url.Values) *Form {
	if f.err != nil {
		return f
	}

	for key, values := range values {
		f.Set(key, values...)
	}

	return f
}

// Multipart creates a MultipartForm using this Form as property base.
func (f *Form) Multipart() *MultipartForm {
	mf := &MultipartForm{Form: *f}
	if f.err != nil {
		return mf
	}

	mf.boundary, mf.err = randomBoundary()
	return mf
}
