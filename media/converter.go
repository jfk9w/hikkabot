package media

import (
	"path/filepath"

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
	Directory string
}

func NewAconvertConverter(config *aconvert.Config, dir string) AconvertConverter {
	return AconvertConverter{aconvert.NewClient(nil, *config), dir}
}

func (a AconvertConverter) Convert(metadata *Metadata, in flu.Readable) (Descriptor, error) {
	format := aconvertMIMEType2TargetFormat[metadata.MIMEType]
	resource := a.newResource(metadata.Size)
	defer resource.Cleanup()
	if err := resource.Pull(in); err != nil {
		return nil, err
	}
	resp, err := a.Client.Convert(resource, make(aconvert.Opts).TargetFormat(format))
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

func (a AconvertConverter) newResource(size int64) Resource {
	if a.Directory == "" {
		return NewMemoryResource(int(size))
	}

	return NewFileResource(filepath.Join(a.Directory, newID()))
}
