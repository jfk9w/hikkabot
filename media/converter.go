package media

import (
	aconvert "github.com/jfk9w-go/aconvert-api"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Converter interface {
	Convert(Type, ReadWrite) (TelegramType, error)
}

var UnsupportedTypeErr = errors.New("unsupported type")

type baseConverter map[Type]TelegramType

func (c baseConverter) AddType(tg TelegramType, types ...Type) baseConverter {
	for _, typ := range types {
		c[typ] = tg
	}
	return c
}

var BaseConverter = baseConverter{}.
	AddType(telegram.Photo, "jpg", "jpeg", "png", "bmp").
	AddType(telegram.Video, "gif", "mp4", "gifv")

func (c baseConverter) Convert(typ Type, _ ReadWrite) (TelegramType, error) {
	if typ, ok := c[typ]; ok {
		return typ, nil
	} else {
		return unknownType, UnsupportedTypeErr
	}
}

type Aconverter struct {
	*aconvert.Client
	types map[Type][2]string
}

func NewAconverter(config aconvert.Config) Aconverter {
	return Aconverter{
		Client: aconvert.NewClient(nil, config),
		types: map[Type][2]string{
			"webm": {"mp4", telegram.Video}},
	}
}

func (a Aconverter) Convert(typ Type, rw ReadWrite) (TelegramType, error) {
	if typ, ok := a.types[typ]; ok {
		resp, err := a.Client.Convert(rw, make(aconvert.Opts).TargetFormat(typ[0]))
		if err != nil {
			return unknownType, err
		}
		err = a.Download(resp.URL(), rw)
		if err != nil {
			return unknownType, err
		}
		return typ[1], nil
	} else {
		return unknownType, UnsupportedTypeErr
	}
}
