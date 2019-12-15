package dvach

import (
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"
)

type query struct {
	*regexp.Regexp
}

func (q *query) MarshalJSON() ([]byte, error) {
	str := ""
	if q.Regexp != nil {
		str = q.String()
	}
	return json.Marshal(str)
}

func (q *query) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return errors.Wrap(err, "on unmarshal")
	}
	if str == "" {
		return nil
	}
	re, err := regexp.Compile(str)
	if err != nil {
		return errors.Wrap(err, "on regexp compilation")
	}
	q.Regexp = re
	return nil
}

func (q *query) String() string {
	if q.Regexp == nil {
		return ""
	}
	return q.Regexp.String()
}
