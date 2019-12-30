package dvach

import (
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"
)

type Query struct {
	*regexp.Regexp
}

func (q Query) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.String())
}

func (q *Query) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return errors.Wrap(err, "unmarshal")
	}
	if str == "" {
		return nil
	}
	re, err := regexp.Compile(str)
	if err != nil {
		return errors.Wrap(err, "compile regexp")
	}
	q.Regexp = re
	return nil
}

func (q *Query) String() string {
	if q.Regexp == nil {
		return ""
	}
	return q.Regexp.String()
}
