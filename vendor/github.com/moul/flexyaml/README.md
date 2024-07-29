# flexyaml

[![CircleCI](https://circleci.com/gh/moul/flexyaml.svg?style=svg)](https://circleci.com/gh/moul/flexyaml)
[![GoDoc](https://godoc.org/github.com/moul/flexyaml?status.svg)](https://godoc.org/github.com/moul/flexyaml)

Flexible yaml is based on http://gopkg.in/yaml.v2 and supports case insensitive keys (!= RFC)

Used in https://github.com/moul/advanced-ssh-config

## Example

```golang
package main

import (
    "fmt"
    "log"

    "github.com/moul/flexyaml"
)

// An example showing how to unmarshal embedded
// structs from YAML.

type StructA struct {
    A string `yaml:"a"`
}

type StructB struct {
    // Embedded structs are not treated as embedded in YAML by default. To do that,
    // add the ",inline" annotation below
    StructA `yaml:",inline"`
    B       string `yaml:"b"`
}

var data = `
a: a string from struct A
B: a string from struct B
`

func main() {
    var b StructB

    err := flexyaml.Unmarshal([]byte(data), &b)
    if err != nil {
        log.Fatal("cannot unmarshal data: %v", err)
    }
    fmt.Println(b.A)
    fmt.Println(b.B)
}

```

## License

MIT
