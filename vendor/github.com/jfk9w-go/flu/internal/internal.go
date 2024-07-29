package internal

import (
	"crypto/md5"
	"fmt"
)

// ID returns md5 hash sum of the value.
func ID(value any) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%p", value))))
}

// Readable returns either the result of String() if value implements fmt.Stringer,
// else first ten symbols of md5 hash sum with value type are returned.
// This is useful for printing values to logs, for example.
func Readable(value any) string {
	if v, ok := value.(fmt.Stringer); ok {
		return v.String()
	} else if v, ok := value.(string); ok {
		return v
	} else {
		return fmt.Sprintf("%T@%s", value, ID(value)[:10])
	}
}
