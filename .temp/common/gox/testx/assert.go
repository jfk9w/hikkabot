package testx

import (
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// Utility type.
type AssertType struct {
	t *testing.T
}

// Create an AssertType instance.
// Usage example:
// 	assert := testx.Assert(t)
//	assert.Equals(5, list.Size())
func Assert(t *testing.T) *AssertType {
	return &AssertType{t}
}

// Assert equality.
func (a *AssertType) Equals(expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		a.FailNotEqual(expected, actual)
	}
}

// Assert int64 equality.
func (a *AssertType) EqualsInt64(expected, actual int64) {
	if expected != actual {
		a.FailNotEqual(expected, actual)
	}
}

// Assert condition is true.
func (a *AssertType) True(value bool) {
	a.Equals(true, value)
}

// Print stack and fail.
func (a *AssertType) FailNotEqual(expected, actual interface{}) {
	debug.PrintStack()
	a.t.Fatal("\nExpected: " + spew.Sdump(expected) + "Actual: " + spew.Sdump(actual))
}
