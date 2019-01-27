package dvach

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/hikkabot/common/httpx"
	"github.com/pkg/errors"
)

const (
	Endpoint = "https://2ch.hk"

	// File types
	Jpeg = 1
	Png  = 2
	Gif  = 4
	Webm = 6
	Mp4  = 10
)

type Config struct {
	Http *httpx.Config `json:"http"`
}

type Error struct {
	Code int    `json:"Code"`
	Err  string `json:"Error"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Err)
}

func ToNum(value string) (Num, bool) {
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return num, true
}

var (
	tz     *time.Location
	tzOnce sync.Once
)

func ToTime(value string) (time.Time, bool) {
	tzOnce.Do(func() {
		loc, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			panic(err)
		}

		tz = loc
	})

	dateRunes := []rune(value)
	date, err := time.ParseInLocation("02/01/06 15:04:05",
		string(dateRunes[:8])+string(dateRunes[12:]), tz)
	if err != nil {
		return time.Time{}, false
	}

	return date, true
}

func ToRef(board Board, numstr string) (Ref, error) {
	num, ok := ToNum(numstr)
	if !ok {
		return Ref{}, errors.Errorf("invalid num: %s", numstr)
	}

	return Ref{numstr, strings.ToLower(board), num}, nil
}
