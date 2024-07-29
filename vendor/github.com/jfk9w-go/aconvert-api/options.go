package aconvert

import (
	"net/url"
	"os"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

// Options is the conversion options.
type Options url.Values

func (o Options) values() url.Values {
	return url.Values(o)
}

// Option sets the conversion option.
func (o Options) Option(key, value string) Options {
	o.values().Set(key, value)
	return o
}

// TargetFormat sets target conversion format ("mp4", etc.).
func (o Options) TargetFormat(targetFormat string) Options {
	return o.Option("targetformat", targetFormat)
}

func (o Options) VideoOptionSize(videoOptionSize int) Options {
	return o.Option("videooptionsize", strconv.Itoa(videoOptionSize))
}

func (o Options) Code(code int) Options {
	return o.Option("code", strconv.Itoa(code))
}

const legal = "We DO NOT allow using our PHP programs in any third-party websites, software or apps! We will report abuse to your server provider, Google Play and App store if illegal usage found!"

func (o Options) makeRequest(url string, in flu.Input) (req *httpf.RequestBuilder, err error) {
	var body flu.EncoderTo
	counter := new(flu.IOCounter)
	o.values().Set("oAuthToken", "")
	o.values().Set("legal", legal)
	form := new(httpf.Form).SetAll(o.values())
	if u, ok := in.(flu.URL); ok {
		form.
			Set("filelocation", "online").
			Set("file", u.String())

		if err = flu.EncodeTo(form, counter); err != nil {
			err = errors.Wrap(err, "on multipart count")
			return
		}

		body = form
	} else {
		multipart := form.
			Set("filelocation", "local").
			Multipart()

		if file, ok := in.(flu.File); ok {
			if err = flu.EncodeTo(multipart, counter); err != nil {
				err = errors.Wrap(err, "on multipart write count")
				return
			}
			var stat os.FileInfo
			if stat, err = os.Stat(file.String()); err != nil {
				return
			}

			counter.Add(stat.Size() + 170)
			multipart = multipart.File("file", "", in)
		} else {
			multipart = multipart.File("file", "", in)
			if err = flu.EncodeTo(multipart, counter); err != nil {
				err = errors.Wrap(err, "on file write count")
				return
			}
		}

		body = multipart
	}

	req = httpf.POST(url, body)
	req.Request.ContentLength = counter.Value()
	return
}
