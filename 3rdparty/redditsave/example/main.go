package main

import (
	"fmt"

	"github.com/jfk9w-go/flu/colf"
)

func main() {
	keys := colf.Keys[string, string](map[string]string{"1": "0", "2": "0"})
	fmt.Printf("%v\n", keys)
}
