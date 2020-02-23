package media

import (
	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
)

type Converter interface {
	Convert(metadata *Metadata, in flu.Readable) (Descriptor, error)
	MIMETypes() []string
}

var aconvertMIMEType2TargetFormat = map[string]string{
	"video/webm": "mp4",
}

type AconvertConverter struct {
	*aconvert.Client
	BufferSpace BufferSpace
}

func NewAconvertConverter(config aconvert.Config, bufferSpace BufferSpace) AconvertConverter {
	return AconvertConverter{aconvert.NewClient(nil, config), bufferSpace}
}

func (a AconvertConverter) Convert(metadata *Metadata, in flu.Readable) (Descriptor, error) {
	format := aconvertMIMEType2TargetFormat[metadata.MIMEType]
	resource := a.BufferSpace.NewResource(metadata.Size)
	defer resource.Cleanup()
	if err := resource.Pull(in); err != nil {
		return nil, err
	}
	resp, err := a.Client.Convert(resource, make(aconvert.Opts).TargetFormat(format))
	if err != nil {
		return nil, err
	}
	if resource, ok := in.(Resource); ok {
		resource.Cleanup()
	}
	return URLDescriptor{
		Client: a.Client.Client,
		URL:    resp.URL(),
	}, nil
}

func (a AconvertConverter) MIMETypes() []string {
	return []string{"video/webm"}
}
