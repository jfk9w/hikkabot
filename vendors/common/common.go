package common

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type Query struct {
	*regexp.Regexp
}

func (q *Query) MatchString(str string) bool {
	if q == nil || q.Regexp == nil {
		return true
	} else {
		return q.Regexp.MatchString(str)
	}
}

func (q *Query) MarshalJSON() ([]byte, error) {
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
	if q == nil {
		return ".*"
	}
	if q.Regexp == nil {
		return ""
	}
	return q.Regexp.String()
}

type PlainSQLBuilder struct {
	SQL       string
	Arguments []interface{}
}

func (p PlainSQLBuilder) ToSQL() (string, []interface{}, error) {
	return p.SQL, p.Arguments, nil
}

type InvalidMediaRef struct {
	Error error
}

func (r InvalidMediaRef) Get(_ context.Context) (format.Media, error) {
	return format.Media{}, r.Error
}

type ResolvedMediaRef struct {
	mimeType string
	input    flu.Input
}

func NewResolvedMediaRef(mimeType string, input flu.Input) ResolvedMediaRef {
	return ResolvedMediaRef{
		mimeType: mimeType,
		input:    input,
	}
}

func (r ResolvedMediaRef) Get(_ context.Context) (format.Media, error) {
	return format.Media{
		MIMEType: r.mimeType,
		Input:    r.input,
	}, nil
}
