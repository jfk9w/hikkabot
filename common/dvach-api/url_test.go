package dvach

import (
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

func Test_ParseUrl(t *testing.T) {
	var assert = testx.Assert(t)
	var url = "https://2ch.hk/abu/res/42375.html"
	var ref, err = ParseUrl(url)
	if err != nil {
		panic(err)
	}

	assert.Equals(Ref{
		Board:     "abu",
		Num:       42375,
		NumString: "42375",
	}, ref)

	url = "https://2ch.hk/abu/res/42375.html#49947"
	ref, err = ParseUrl(url)
	if err != nil {
		panic(err)
	}

	assert.Equals(Ref{
		Board:     "abu",
		Num:       49947,
		NumString: "49947",
	}, ref)
}
