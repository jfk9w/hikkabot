package media

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var sizeRegexp = regexp.MustCompile(`^(\d+)([kmgt])?$`)

const (
	b  Size = 1
	kb      = 1 << 10
	mb      = 1 << 20
	gb      = 1 << 30
	tb      = 1 << 40
)

type Size int64

func (s *Size) UnmarshalYAML(node *yaml.Node) error {
	match := sizeRegexp.FindStringSubmatch(strings.ToLower(node.Value))
	if len(match) < 2 {
		return errors.Errorf(`expected expression matching %s`, sizeRegexp.String())
	}

	amount, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return err
	}

	unit := b
	if len(match) == 3 {
		switch match[2] {
		case "k":
			unit = kb
		case "m":
			unit = mb
		case "g":
			unit = gb
		case "t":
			unit = tb
		}
	}

	*s = unit * Size(amount)
	return nil
}

func (s Size) MarshalYAML() (any, error) {
	return strconv.FormatInt(int64(s), 10), nil
}

func (s Size) String() string {
	size := int64(s)
	switch {
	case size >= tb:
		return fmt.Sprintf("%dT", size/tb)
	case size >= gb:
		return fmt.Sprintf("%dG", size/gb)
	case size >= mb:
		return fmt.Sprintf("%dM", size/mb)
	case size >= kb:
		return fmt.Sprintf("%dK", size/kb)
	default:
		return fmt.Sprintf("%d", size)
	}
}
