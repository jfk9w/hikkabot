package util

import "github.com/jfk9w-go/flu"

func Read(resource flu.ReadResource, body flu.BodyReader) error {
	reader, err := resource.Read()
	if err != nil {
		return err
	}

	defer reader.Close()
	return body.Read(reader)
}

func ReadJSON(filepath string, value interface{}) {
	Check(Read(flu.NewFileSystemResource(filepath), flu.JSON(value)))
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}
