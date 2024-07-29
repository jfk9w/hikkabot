package httpf

import (
	"fmt"
)

// StatusCodeError is returned when response status code does not match expected.
type StatusCodeError struct {
	// StatusCode is the response status code.
	StatusCode int
	// Status is the response status.
	Status string
}

func (e StatusCodeError) Error() string {
	if e.Status != "" {
		return e.Status
	}

	return fmt.Sprint(e.StatusCode)
}

// ContentTypeError is returned when response Content-Type does not match the expected one.
type ContentTypeError string

func (e ContentTypeError) Error() string {
	return fmt.Sprintf("invalid body type: %s", string(e))
}

// VarargsLengthError is returned when length of passed vararg is not even.
type VarargsLengthError int

func (e VarargsLengthError) Error() string {
	return fmt.Sprintf("key-value pairs array length must be even, got %d", e.Length())
}

func (e VarargsLengthError) Length() int {
	return int(e)
}
