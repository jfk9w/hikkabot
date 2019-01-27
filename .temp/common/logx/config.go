package logx

import (
	"os"

	"github.com/jfk9w-go/hikkabot/common/gox/jsonx"
)

type (
	// A single logger config. Defines the minimal level and outputs.
	LoggerConfig struct {

		// Level is the lowest log level to be printed.
		Level string `json:"level"`

		// Output is the list of output log files.
		// Two special values exist:
		// stdout - standard output
		// stderr - standard error output
		Output []string `json:"output"`
	}

	// Logger factory config.
	Config struct {

		// Default configuration is used when a logger name is not recognized.
		Default LoggerConfig `json:"default"`

		// Custom contains logger-specific configurations resolved by name.
		Custom map[string]LoggerConfig `json:"custom"`
	}

	embeddedConfig struct {
		Logging *Config `json:"logx"`
	}
)

var defaultcfg = Config{
	Default: LoggerConfig{
		Level:  "debug",
		Output: []string{"stdout"},
	},
}

func config() Config {
	path := os.Getenv("LOGX")
	if len(path) == 0 {
		return defaultcfg
	}

	ec := &embeddedConfig{}
	if err := jsonx.ReadFile(path, ec); err != nil {
		panic(err)
	}

	if ec.Logging == nil {
		return defaultcfg
	}

	return *ec.Logging
}
