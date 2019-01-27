package jsonx

import (
	"testing"
	"time"

	"github.com/jfk9w-go/hikkabot/common/gox/testx"
)

type durationTest struct {
	Millisecond Duration `json:"millisecond"`
	Second      Duration `json:"second"`
	Minute      Duration `json:"minute"`
	Hour        Duration `json:"hour"`
	Day         Duration `json:"day"`
	Month       Duration `json:"month"`
}

func TestDuration(t *testing.T) {
	assert := testx.Assert(t)
	val := new(durationTest)
	if err := ReadFile("./testdata/duration.json", val); err != nil {
		t.Fatal(err)
	}

	assert.Equals(1*time.Millisecond, val.Millisecond.Duration())
	assert.Equals(2*time.Second, val.Second.Duration())
	assert.Equals(30*time.Minute, val.Minute.Duration())
	assert.Equals(5*time.Hour, val.Hour.Duration())
	assert.Equals(7*24*time.Hour, val.Day.Duration())
	assert.Equals(3*30*24*time.Hour, val.Month.Duration())
}
