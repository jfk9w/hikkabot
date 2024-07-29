package apfel

import (
	"context"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/pkg/errors"
)

type (
	LogfRule struct {
		Match  string     `yaml:"match" format:"regex" doc:"Defines a regex which is used for matching logger names." example:"^apfel"`
		Level  logf.Level `yaml:"level,omitempty" doc:"Level threshold. Messages with level below the threshold will not be logged. Defaults to the global default." enum:"trace,debug,info,warn,error,panic,silent" example:"info"`
		Output string     `yaml:"output,omitempty" doc:"Log output. Either a file path or one of examples. Defaults to the global default." examples:"stdout,stderr"`
	}

	LogfConfig struct {
		Level  logf.Level `yaml:"level,omitempty" doc:"Default level threshold. Messages with level below the threshold will not be logged." enum:"trace,debug,info,warn,error,panic,silent" default:"info"`
		Output string     `yaml:"output,omitempty" doc:"Default log output. Either a file path or one of examples." examples:"stdout,stderr" default:"stderr"`
		Rules  []LogfRule `yaml:"rules,omitempty" doc:"Loggers are matched to rules in the order of appearance. If no rules match, the settings from this object are used."`
	}

	// LogfContext is the logf application configuration interface.
	LogfContext interface{ LogfConfig() LogfConfig }
)

// Logf configures logging via logf package.
// Implements Mixin interface.
type Logf[C LogfContext] struct{}

func (m Logf[C]) String() string {
	return "logf"
}

func (m *Logf[C]) Include(ctx context.Context, app MixinApp[C]) error {
	config := app.Config().LogfConfig()
	rules := make([]logfCompiledRule, 0, len(config.Rules)+1)
	for _, r := range config.Rules {
		match, err := regexp.Compile(r.Match)
		if err != nil {
			return errors.Wrapf(err, "compile logf prefix regexp [%s]", r.Match)
		}

		rule := logfCompiledRule{
			match:  match,
			level:  r.Level,
			output: logfOutput(r.Output),
		}

		if rule.level == 0 {
			rule.level = config.Level
		}

		if rule.output == "" {
			rule.output = logfOutput(config.Output)
		}

		rules = append(rules, rule)
	}

	rules = append(rules, logfCompiledRule{level: config.Level, output: logfOutput(config.Output)})
	logf.ResetFactory(func(name string, defaultLogger logf.Interface) logf.Interface {
		for _, r := range rules {
			if r.match != nil && !r.match.MatchString(name) {
				continue
			}

			writer, err := r.output.Writer()
			if err != nil {
				defaultLogger.Errorf(nil, "unable to open log output for %s at %s, using default: %v", name, r.output, err)
				return defaultLogger
			}

			if err := app.Manage(ctx, writer); err != nil {
				defaultLogger.Errorf(nil, "log output for %s at %s is unmanaged due to (using default): %v", name, r.output, err)
				return defaultLogger
			}

			logger := &logf.BareAdapter{Bare: logf.NewStdLogger(name, log.Flags()|log.Ldate|log.Ltime, writer)}
			logger.SetLevel(r.level)
			return logger
		}

		return defaultLogger
	})

	return nil
}

type logfOutput string

func (o logfOutput) Writer() (io.Writer, error) {
	switch o {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		return flu.File(o).Writer()
	}
}

type logfCompiledRule struct {
	match  *regexp.Regexp
	level  logf.Level
	output logfOutput
}
