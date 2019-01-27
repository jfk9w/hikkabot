package logx

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 37

	defaultTimeFormat = "2006-01-02 15:04:05.000"
	templateColored   = "\x1b[%dm%s\x1b[0m"
)

var (
	spewfmt = &spew.ConfigState{
		Indent:                  "  ",
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		DisableMethods:          true,
		DisablePointerMethods:   true,
	}

	levels = map[logrus.Level]string{
		logrus.PanicLevel: "PANIC",
		logrus.FatalLevel: "FATAL",
		logrus.ErrorLevel: "ERROR",
		logrus.WarnLevel:  "WARN ",
		logrus.InfoLevel:  "INFO ",
		logrus.DebugLevel: "DEBUG",
	}
)

type format struct {
	name string
}

func (f *format) Format(entry *logrus.Entry) ([]byte, error) {
	sb := &strings.Builder{}
	sb.WriteString(entry.Time.Format(defaultTimeFormat))
	sb.WriteRune(' ')
	sb.WriteString(levelColored(entry.Level))
	sb.WriteString(" [")
	sb.WriteString(f.name)
	sb.WriteString("] ")
	sb.WriteString(entry.Message)
	if last, _ := utf8.DecodeLastRune([]byte(entry.Message)); last != '\n' {
		sb.WriteRune('\n')
	}

	for key, value := range entry.Data {
		sb.WriteString(key)
		sb.WriteString(": ")

		dumped := spewfmt.Sdump(value)
		sb.WriteString(dumped)
		if last, _ := utf8.DecodeLastRune([]byte(dumped)); last != '\n' {
			sb.WriteRune('\n')
		}
	}

	return []byte(sb.String()), nil
}

func levelColored(l logrus.Level) string {
	var color int
	switch l {
	case logrus.InfoLevel:
		color = green
	case logrus.WarnLevel:
		color = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		color = red
	default:
		color = blue
	}

	return fmt.Sprintf(templateColored, color, levels[l])
	//return levels[l]
}
