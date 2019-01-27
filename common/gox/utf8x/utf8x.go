// Package utf8x contains various useful utf8 wrappers and utilities
package utf8x

import (
	"unicode/utf8"

	"github.com/jfk9w-go/hikkabot/common/gox/mathx"
)

// Check if the first rune in the string is the specified rune
func IsFirst(str string, r rune) bool {
	if len(str) == 0 {
		return false
	}

	first, _ := utf8.DecodeRune([]byte(str))
	return first == r
}

// Returns n first runes with suffix if the source string is longer than n
func Head(str string, n int, suffix string) string {
	runes := []rune(str)
	if len(runes) > n {
		return string(runes[:n]) + suffix
	}

	return str
}

// Returns the length of the string rune array
func Size(str string) int {
	runes := []rune(str)
	return len(runes)
}

// Slice the string rune-wise
func Slice(str string, a, b int) string {
	if size := Size(str); size > 0 {
		a, b = bounds(size, a, b)
	} else {
		return ""
	}

	runes := []rune(str)
	return string(runes[a:b])
}

// Index of the first rune in the string equal to the specifed rune
func IndexOf(str string, r rune, a, b int) int {
	if size := Size(str); size > 0 {
		a, b = bounds(size, a, b)
	} else {
		return -1
	}

	runes := []rune(str)
	for i := a; i < b; i++ {
		if runes[i] == r {
			return i
		}
	}

	return -1
}

// Index of the last rune in the string equal to the specified rune
func LastIndexOf(str string, r rune, a, b int) int {
	if size := Size(str); size > 0 {
		a, b = bounds(size, a, b)
	} else {
		return -1
	}

	runes := []rune(str)
	for i := b - 1; i >= a; i-- {
		if runes[i] == r {
			return i
		}
	}

	return -1
}

func bounds(size, a, b int) (int, int) {
	a = mathx.MaxInt(a, 0)
	b = mathx.MinInt(b, size)

	for a >= size {
		a -= size
	}

	for b <= 0 {
		b += size
	}

	if b < a {
		a, b = b, a
	}

	return a, b
}
