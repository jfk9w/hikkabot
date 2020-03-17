package feed

import "github.com/jfk9w-go/flu"

type RawData interface {
	Marshal(interface{}) RawData
	Unmarshal(interface{})
	Bytes() []byte
}

type JSON flu.Buffer

func NewRawData(bytes []byte) RawData {
	buf := flu.NewBuffer()
	if bytes != nil {
		if err := flu.Copy(flu.Bytes(bytes), buf); err != nil {
			panic(err)
		}
	}
	return JSON(buf)
}

func (j JSON) Marshal(value interface{}) RawData {
	if err := flu.EncodeTo(flu.JSON{value}, flu.Buffer(j)); err != nil {
		panic(err)
	}
	return j
}

func (j JSON) Unmarshal(value interface{}) {
	if err := flu.DecodeFrom(flu.Buffer(j), flu.JSON{value}); err != nil {
		panic(err)
	}
}
