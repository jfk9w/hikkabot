package api

import (
	"strconv"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

type params = httpx.Params

func arr(vals ...string) []string {
	return vals
}

func ints(value int) []string {
	if value == 0 {
		return arr()
	}

	return arr(strconv.Itoa(value))
}

func bools(value bool) []string {
	if value {
		return arr("true")
	}

	return arr()
}

func strs(value string) []string {
	if value == "" {
		return arr()
	}

	return arr(value)
}

type request struct {
	Method string
	Params params
}
