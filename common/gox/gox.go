// Package gox provides useful utilities and wrappers
// for built-in functionality as well as third-party libraries
package gox

func Check(err error) {
	if err != nil {
		panic(err)
	}
}
