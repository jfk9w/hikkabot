package logx

import (
	"io"
	"os"
	"sync"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/sirupsen/logrus"
)

type internal struct {
	config  Config
	loggers *sync.Map
}

type logger struct {
	Ptr
	sync.Once
}

func (log *logger) init(name string, config LoggerConfig) {
	log.Ptr = logrus.New()

	var err error
	if log.Level, err = logrus.ParseLevel(config.Level); err != nil {
		panic(err)
	}

	writers := make([]io.Writer, len(config.Output))
	for i, path := range config.Output {
		switch path {
		case "stdout":
			writers[i] = os.Stdout

		case "stderr":
			writers[i] = os.Stderr

		default:
			path, err := fsx.Path(path)
			if err != nil {
				panic(err)
			}

			if err := fsx.EnsureParent(path); err != nil {
				panic(err)
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				panic(err)
			}

			writers[i] = file
		}
	}

	log.Out = io.MultiWriter(writers...)
	log.Formatter = &format{name}
}

func (obj *internal) get(name string) Ptr {
	entry, _ := obj.loggers.LoadOrStore(name, new(logger))
	def := entry.(*logger)
	def.Do(func() {
		config, ok := obj.config.Custom[name]
		if !ok {
			config = obj.config.Default
		}

		def.init(name, config)
	})

	return def.Ptr
}
