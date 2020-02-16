package media

import (
	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
)

type Converter interface {
	Convert(mimeType string, in flu.Readable) (Descriptor, error)
	MIMETypes() []string
}

var aconvertMIMEType2TargetFormat = map[string]string{
	"video/webm": "mp4",
}

type AconvertConverter struct {
	*aconvert.Client
}

func NewAconvertConverter(config *aconvert.Config) AconvertConverter {
	return AconvertConverter{aconvert.NewClient(nil, *config)}
}

func (a AconvertConverter) Convert(mimeType string, in flu.Readable) (Descriptor, error) {
	format := aconvertMIMEType2TargetFormat[mimeType]
	resp, err := a.Client.Convert(in, make(aconvert.Opts).TargetFormat(format))
	if err != nil {
		return nil, err
	}
	return URLDescriptor{
		Client: a.Client.Client,
		URL:    resp.URL(),
	}, nil
}

func (a AconvertConverter) MIMETypes() []string {
	return []string{"video/webm"}
}
