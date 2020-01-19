package feed

import "github.com/jfk9w-go/flu"

type RawData interface {
	Marshal(interface{})
	Unmarshal(interface{})
	Bytes() []byte
}

type JSON flu.Buffer

func NewRawData() RawData {
	return JSON(flu.NewBuffer())
}

func (j JSON) Marshal(value interface{}) {
	var in flu.Readable
	if bytes, ok := value.([]byte); ok {
		in = flu.Bytes(bytes)
	} else {
		in = flu.PipeOut(flu.JSON(value))
	}
	if err := flu.Copy(in, flu.Buffer(j)); err != nil {
		panic(err)
	}
}

func (j JSON) Unmarshal(value interface{}) {
	if err := flu.Read(flu.Buffer(j), flu.JSON(value)); err != nil {
		panic(err)
	}
}
