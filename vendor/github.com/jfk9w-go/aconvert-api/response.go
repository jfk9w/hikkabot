package aconvert

import (
	"io"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// Response is the aconvert.com conversion response.
type Response struct {
	// Server is the ID of the server which performed the conversion.
	Server string `json:"server"`
	// Filename is the resulting file name.
	Filename string `json:"filename"`
	// State is the conversion state.
	State string `json:"state"`
	// Result is sometimes returned instead of State (or vice-versa).
	Result string `json:"result"`

	data string
	host string
}

func (r *Response) DecodeFrom(reader io.Reader) error {
	var buf flu.ByteBuffer
	if _, err := flu.Copy(flu.IO{R: reader}, &buf); err != nil {
		return err
	}

	r.data = buf.String()

	if err := flu.DecodeFrom(&buf, flu.JSON(r)); err != nil {
		return errors.Wrapf(err, "failed to decode [%s]: %v", r, err)
	}

	if r.State != "SUCCESS" {
		return errors.Errorf("state is %s, not SUCCESS (%s)", r.State, r)
	}

	r.host = host(r.Server)
	return nil
}

func (r *Response) String() string {
	if r == nil {
		return "<nil>"
	}

	return r.data
}

// URL returns converted file URL.
func (r *Response) URL() string {
	return r.host + "/convert/p3r68-cdx67/" + r.Filename
}
