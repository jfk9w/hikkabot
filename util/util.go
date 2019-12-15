package util

import "github.com/jfk9w-go/flu"

func Read(resource flu.ResourceReader, body flu.DecoderFrom) error {
	reader, err := resource.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()
	return body.DecodeFrom(reader)
}

func ReadJSON(filepath string, value interface{}) {
	Check(Read(flu.File(filepath), flu.JSON(value)))
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}
