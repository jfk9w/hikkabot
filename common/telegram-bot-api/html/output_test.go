package html

import (
	"testing"

	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

const LoremIpsum = "Lorem ipsum dolor sit amet, " +
	"consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
	"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. " +
	"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. " +
	"Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

func TestOutput_Fits(t *testing.T) {
	var assert = testx.Assert(t)
	var limitedOutput = NewOutput(2, 30)
	assert.Equals(true, limitedOutput.Fits(LoremIpsum[:1]))
	assert.Equals(true, limitedOutput.Fits(LoremIpsum[:30]))
	assert.Equals(false, limitedOutput.Fits(LoremIpsum[:31]))
}

func TestOutput_Break(t *testing.T) {
	var assert = testx.Assert(t)
	assert.Equals(27, NewOutput(2, 30).Break(LoremIpsum))
}

func TestOutput_AppendText(t *testing.T) {
	var assert = testx.Assert(t)
	assert.Equals([]string{
		"Lorem ipsum dolor sit amet,",
		"consectetur adipiscing elit,",
		"sed do eiusmod tempor",
		"incididunt ut labore et",
		"dolore magna aliqua. Ut enim",
		"ad minim veniam, quis",
	}, NewOutput(6, 30).AppendText(LoremIpsum).Flush())
}
