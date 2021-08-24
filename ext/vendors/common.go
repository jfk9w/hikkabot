package vendors

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
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

type InvalidMediaRef struct {
	Error error
}

func (r InvalidMediaRef) Get(_ context.Context) (*media.Value, error) {
	return nil, r.Error
}

type ResolvedMediaRef struct {
	mimeType string
	input    flu.Input
}

func NewResolvedMediaRef(mimeType string, input flu.Input) *ResolvedMediaRef {
	return &ResolvedMediaRef{
		mimeType: mimeType,
		input:    input,
	}
}

func (r *ResolvedMediaRef) Get(_ context.Context) (*media.Value, error) {
	return &media.Value{
		MIMEType: r.mimeType,
		Input:    r.input,
	}, nil
}

var (
	tagRegexp  = regexp.MustCompile(`<.*?>`)
	junkRegexp = regexp.MustCompile(`(?i)[^\wа-яё_]`)
)

func Hashtag(str string) string {
	str = html.UnescapeString(str)
	str = tagRegexp.ReplaceAllString(str, "")
	fields := strings.Fields(str)
	for i, field := range fields {
		fields[i] = strings.Title(junkRegexp.ReplaceAllString(field, ""))
	}
	str = strings.Join(fields, "")
	tag := utf8string.NewString(str)
	if tag.RuneCount() > 25 {
		return "#" + tag.Slice(0, 25)
	}
	return "#" + tag.String()
}
