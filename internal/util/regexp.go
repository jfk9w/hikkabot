package util

import (
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"
)

type Regexp struct {
	*regexp.Regexp
}

func (re Regexp) MatchString(str string) bool {
	if re.Regexp == nil {
		return true
	} else {
		return re.Regexp.MatchString(str)
	}
}

func (re Regexp) MarshalJSON() ([]byte, error) {
	return json.Marshal(re.String())
}

func (re *Regexp) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return errors.Wrap(err, "unmarshal")
	}
	if str == "" {
		return nil
	}
	regexp, err := regexp.Compile(str)
	if err != nil {
		return errors.Wrap(err, "compile regexp")
	}
	re.Regexp = regexp
	return nil
}

func (re Regexp) String() string {
	if re.Regexp == nil {
		return ""
	}

	return re.Regexp.String()
}
