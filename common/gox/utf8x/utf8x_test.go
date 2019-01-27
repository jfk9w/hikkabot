package utf8x

import (
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

func TestIsFirst(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals(false, IsFirst("", 'ю'))
	assert.Equals(true, IsFirst("ю", 'ю'))
	assert.Equals(true, IsFirst("юэ", 'ю'))
}

func TestHead(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals("", Head("", 10, "..."))
	assert.Equals("абвгдежзикл", Head("абвгдежзикл", 11, "..."))
	assert.Equals("абвгдежзик...", Head("абвгдежзикл", 10, "..."))
	assert.Equals("абвгд...", Head("абвгдежзикл", 5, "..."))
}

func TestSize(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals(0, Size(""))
	assert.Equals(1, Size("ю"))
	assert.Equals(11, Size("абвгд ежзик"))
}

func TestSlice(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals("", Slice("", 0, 0))
	assert.Equals("ю", Slice("ю", 0, 1))
	assert.Equals("юэ", Slice("юэа", 0, 2))
	assert.Equals("эа", Slice("юэа", 1, 3))
}

func TestIndexOf(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals(-1, IndexOf("", 'ю', 0, 0))
	assert.Equals(0, IndexOf("ю", 'ю', 0, 0))
	assert.Equals(3, IndexOf("абвгд ежзик", 'г', 0, 0))
	assert.Equals(-1, IndexOf("абвгд ежзик", 'г', 0, 3))
	assert.Equals(-1, IndexOf("абвгд ежзик", 'г', 4, 0))
	assert.Equals(7, IndexOf("абвгд ежзик", 'ж', 4, 0))
}

func TestLastIndexOf(t *testing.T) {
	assert := testx.Assert(t)
	assert.Equals(-1, LastIndexOf("", 'ю', 0, 0))
	assert.Equals(0, LastIndexOf("ю", 'ю', 0, 0))
	assert.Equals(4, LastIndexOf("габвгд ежзик", 'г', 0, 0))
	assert.Equals(0, LastIndexOf("габвгд ежзик", 'г', 0, 4))
	assert.Equals(-1, LastIndexOf("габвгд ежзик", 'г', 5, 0))
	assert.Equals(8, LastIndexOf("жабвгд ежзик", 'ж', 4, 0))
}
