package media

import (
	aconvert "github.com/jfk9w-go/aconvert-api"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Converter interface {
	Convert(string, SizeAwareReadable) (SizeAwareReadable, telegram.MediaType, error)
}

var UnsupportedTypeErr = errors.New("unsupported type")

type FormatSupport map[string]telegram.MediaType

func (sup FormatSupport) AddFormats(typ telegram.MediaType, formats ...string) FormatSupport {
	for _, format := range formats {
		sup[format] = typ
	}
	return sup
}

func (sup FormatSupport) Convert(format string, in SizeAwareReadable) (out SizeAwareReadable, typ telegram.MediaType, err error) {
	var ok bool
	typ, ok = sup[format]
	if ok {
		out = in
	} else {
		err = UnsupportedTypeErr
	}
	return
}

var SupportedFormats = FormatSupport{}.
	AddFormats(telegram.Photo, "jpg", "jpeg", "png", "bmp").
	AddFormats(telegram.Video, "gif", "mp4", "gifv")

type Aconverter struct {
	*aconvert.Client
	formats map[string][2]string
}

func NewAconverter(config aconvert.Config) Aconverter {
	return Aconverter{
		Client: aconvert.NewClient(nil, config),
		formats: map[string][2]string{
			"webm": {"mp4", telegram.Video}},
	}
}

func (a Aconverter) Convert(format string, in SizeAwareReadable) (out SizeAwareReadable, typ telegram.MediaType, err error) {
	if supformat, ok := a.formats[format]; ok {
		var resp *aconvert.Response
		resp, err = a.Client.Convert(in, make(aconvert.Opts).TargetFormat(supformat[0]))
		if err != nil {
			return
		}
		out = &HTTPRequestReadable{
			Request: a.Client.NewRequest().
				Resource(resp.URL()).
				GET(),
		}
		typ = supformat[1]
	} else {
		err = UnsupportedTypeErr
	}
	return
}
