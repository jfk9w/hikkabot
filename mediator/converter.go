package mediator

import (
	"log"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type ConvertedRequest struct {
	Request
	telegram.MediaType
}

type Converter interface {
	Convert(flu.Readable, *Metadata) (*ConvertedRequest, error)
}

var ErrUnsupportedType = errors.New("unsupported type")

type FormatSupport map[string]telegram.MediaType

func (sup FormatSupport) AddFormats(typ telegram.MediaType, formats ...string) FormatSupport {
	for _, format := range formats {
		sup[format] = typ
	}
	return sup
}

func (sup FormatSupport) Convert(in flu.Readable, metadata *Metadata) (*ConvertedRequest, error) {
	if typ, ok := sup[metadata.Format]; ok {
		return &ConvertedRequest{
			Request: &DoneRequest{
				Readable:  in,
				Metadata_: metadata,
			},
			MediaType: typ,
		}, nil
	} else {
		return nil, ErrUnsupportedType
	}
}

var SupportedFormats = FormatSupport{}.
	AddFormats(telegram.Photo, "jpg", "jpeg", "png", "bmp").
	AddFormats(telegram.Video, "gif", "mp4", "gifv")

type Aconverter struct {
	*aconvert.Client
	formatTypes map[string][2]string
}

func NewAconverter(client *aconvert.Client) Aconverter {
	return Aconverter{
		Client: client,
		formatTypes: map[string][2]string{
			"webm": {"mp4", telegram.Video}},
	}
}

func (a Aconverter) Convert(in flu.Readable, metadata *Metadata) (*ConvertedRequest, error) {
	if formatType, ok := a.formatTypes[metadata.Format]; ok {
		resp, err := a.Client.Convert(flu.URL(metadata.URL), make(aconvert.Opts).TargetFormat(formatType[0]))
		if err != nil {
			log.Printf("Failed to convert %s with aconvert as URL", metadata.URL)
			resp, err = a.Client.Convert(in, make(aconvert.Opts).TargetFormat(formatType[0]))
			if err != nil {
				return nil, errors.Wrap(err, "aconvert")
			}
		}
		return &ConvertedRequest{
			Request: &HTTPRequest{
				URL:    resp.URL(),
				Format: formatType[0],
			},
			MediaType: formatType[1],
		}, nil
	} else {
		return nil, ErrUnsupportedType
	}
}
