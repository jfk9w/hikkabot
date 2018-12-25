package jsonx

import (
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var durationRegexp = regexp.MustCompile(`([0-9]+)\s*([a-z]+)`)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	str, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	groups := durationRegexp.FindStringSubmatch(str)
	if len(groups) != 3 {
		return errors.Errorf("invalid duration format: %s", data)
	}

	value, _ := strconv.ParseInt(string(groups[1]), 10, 64)

	var unit time.Duration
	switch groups[2] {
	case "ms", "milli", "millis", "millisecond", "milliseconds":
		unit = time.Millisecond
	case "s", "second", "seconds":
		unit = time.Second
	case "min", "minute", "minutes":
		unit = time.Minute
	case "hr", "hrs", "hour", "hours":
		unit = time.Hour
	case "d", "day", "days":
		unit = 24 * time.Hour
	case "month", "months":
		unit = 30 * 24 * time.Hour
	default:
		return errors.Errorf("invalid duration unit: %s", groups[2])
	}

	*d = Duration(time.Duration(value) * unit)
	return nil
}

func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}
