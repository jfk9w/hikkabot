package mediator

import aconvert "github.com/jfk9w-go/aconvert-api"

type Size struct {
	Bytes     int64
	Kilobytes int64
	Megabytes int64
}

func (s *Size) Value(defaultValue int64) int64 {
	if s == nil {
		return defaultValue
	} else {
		return s.Megabytes<<20 + s.Kilobytes<<10 + s.Bytes
	}
}

type Config struct {
	Concurrency      int
	Aconvert         *aconvert.Config
	MinSize, MaxSize *Size
	Buffer           bool
}
