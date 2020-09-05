package common

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

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

type StringSet map[string]bool

func (s StringSet) Has(key string) bool {
	return s[key]
}

func (s StringSet) Add(key string) {
	s[key] = true
}

func (s StringSet) ForEach(fun func(key string) bool) {
	for k, v := range s {
		if !v {
			continue
		}

		if !fun(k) {
			return
		}
	}
}

func (s StringSet) Delete(key string) {
	s[key] = false
}

func (s StringSet) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteRune('[')
	first := true
	s.ForEach(func(key string) bool {
		if first {
			first = false
		} else {
			b.WriteString(", ")
		}

		b.WriteRune('"')
		b.WriteString(key)
		b.WriteRune('"')
		return true
	})

	b.WriteRune(']')
	return []byte(b.String()), nil
}

func (s StringSet) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str[0] != '[' {
		return errors.New("expected array start")
	}

	var b strings.Builder
	var write bool
	for _, c := range str[1:] {
		if c == '"' {
			write = !write
			if !write {
				s[b.String()] = true
				b.Reset()
			}
		} else if write {
			b.WriteRune(c)
		}
	}

	return nil
}

func (s StringSet) Copy() StringSet {
	copy := make(StringSet, len(s))
	s.ForEach(func(key string) bool {
		copy.Add(key)
		return true
	})

	return copy
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
