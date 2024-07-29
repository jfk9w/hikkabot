package apfel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu/apfel/schema"
	"github.com/jfk9w-go/flu/colf"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel/internal"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

var DefaultConfigSourceHelpTemplate = `
  Available configuration codecs: json, yml, gob

  Configuration values can be provided either as (in the order of override):

    1. Standard input (with --config.stdin=<codec> option) and files (with --config.file=<filepath> with codec
       detection by filename extension [.yml, .json, .gob]).
    2. Environment variables with '%s' prefix. For example, if you have configuration '{"service":{"test":{"enabled":true}}}',
       that may be expressed in environment variable as '%sservice_test_enabled=true'.
       Please note that environment variable must match case of the configuration property name,
       so, for example, serviceReloader.connectionTimeout is mapped to %sserviceReloader_connectionTimeout environment variable.  
    3. Command-line arguments. The example from the previous point could be passed as CLI arguments like 
       '--service.test.enabled=true' or ('--service.test.enabled' in case of boolean value).

  It is not mandatory to use only one configuration source, you can split files and options and pass them in as you like.

`

// DefaultConfigSource creates a generally applicable ConfigSource instance which reads
// configuration properties from stdin (with --config.stdin option), file (with --config.file=<filepath> option,
// the codec is automatically resolved by filename extension, see ExtensionCodecs),
// environment variables (with appName_ prefix with all hyphens replaced with underscores),
// and direct CLI arguments (like --service.timeout=10s).
// See DefaultConfigSourceHelpTemplate for more info.
func DefaultConfigSource(appName string) ConfigSource {
	environPrefix := strings.Replace(appName, "-", "_", -1) + "_"
	return &ConfigSources{
		Name:     appName,
		HelpText: fmt.Sprintf(DefaultConfigSourceHelpTemplate, environPrefix, environPrefix, environPrefix),
		Sources: []ConfigSource{
			&InputSource{
				Args:        os.Args[1:],
				Stdin:       flu.IO{R: os.Stdin},
				FileOption:  "config.file",
				StdinOption: "config.stdin",
			},
			Environ(os.Environ(), environPrefix),
			Arguments(os.Args[1:]),
		},
	}
}

// Default creates a value of type C
// and fills it with default values based on resolved JSON schema.
func Default[C any]() C {
	ctx := context.Background()
	var config C
	schema, err := schema.Generate(reflect.TypeOf(config))
	if err != nil {
		logf.Panicf(ctx, "generate config schema: %+v", err)
	}

	if err := schema.ApplyDefaults(&config, false); err != nil {
		logf.Panicf(ctx, "apply pre config defaults: %v", err)
	}

	if err := schema.ApplyDefaults(&config, true); err != nil {
		logf.Panicf(ctx, "apply post config defaults: %v", err)
	}

	return config
}

// ConfigSource implementations read the configuration properties into internal.AnyMap
// from a configuration source.
type ConfigSource interface {
	ReadInto(ctx context.Context, values internal.AnyMap) error
}

// ConfigSourceFunc is the ConfigSource functional adapter.
type ConfigSourceFunc func(ctx context.Context, values internal.AnyMap) error

func (f ConfigSourceFunc) ReadInto(ctx context.Context, values internal.AnyMap) error {
	return f(ctx, values)
}

// KeyValueSource is used to read configuration as key-value pairs from a collection of string values.
// Nested option keys are joined using the PathSeparator.
// Type specification is performed on keys as well as values (and nested objects).
// Please note that this Parser does not support options with array values.
// Having a nested and simple value for the same key (for example, APP_OPTIONS=a and APP_OPTIONS_DEBUG=true)
// will cause a warning and preference of the option with the longest path
// (converting into Map and overwriting all of its parents).
type KeyValueSource struct {

	// Prefix is an optional prefix for configuration options.
	// If set, only the options with this prefix are read,
	// and the prefix is stripped from option key.
	Prefix string

	// PathSeparator is used for separating nested configuration keys (for example, service.instance.name contains . as a separator).
	PathSeparator string

	// Separator is a key-value separator.
	Separator string

	// Lines is a collection of strings containing key-value pairs.
	Lines []string

	// Except is a collection of option regexes which should be ignored when reading configuration from Lines.
	Except colf.Set[string]
}

func (s *KeyValueSource) ReadInto(ctx context.Context, values internal.AnyMap) error {
main:
	for _, line := range s.Lines {
		if !strings.HasPrefix(line, s.Prefix) {
			continue
		}

		line = line[len(s.Prefix):]
		equals := strings.Index(line, s.Separator)
		var key, value string
		if equals > 0 {
			key, value = line[:equals], line[equals+1:]
		} else {
			key, value = line, "true"
		}

		for except := range s.Except {
			if matched, err := regexp.MatchString(except, key); err != nil {
				return errors.Wrapf(err, "match regexp [%s]", except)
			} else if matched {
				continue main
			}
		}

		keyTokens := strings.Split(key, s.PathSeparator)
		keyTokensLastIdx := len(keyTokens) - 1
		config := values
		for i, keyToken := range keyTokens {
			if keyToken == "" {
				break
			}

			//keyToken = strings.ToLower(keyToken)
			typedKeyToken, _ := internal.SpecifyType(keyToken)
			if i == keyTokensLastIdx {
				if entryValue, ok := config[typedKeyToken]; ok {
					if _, ok := entryValue.(internal.AnyMap); ok {
						logf.Get(s).Warnf(ctx, "discarding key [%s] due to type incompatibility", key)
						continue
					}
				}

				config[typedKeyToken], _ = internal.SpecifyType(value)
			} else {
				var entryMapValue internal.AnyMap
				if entryValue, ok := config[typedKeyToken]; ok {
					if entryMapValue, ok = entryValue.(internal.AnyMap); !ok {
						logf.Get(s).Printf(ctx, "overriding parent for key [%s] as object", key)
						entryMapValue = make(internal.AnyMap)
						config[typedKeyToken] = entryMapValue
					}
				} else {
					entryMapValue = make(internal.AnyMap)
					config[typedKeyToken] = entryMapValue
				}

				config = entryMapValue
			}
		}
	}

	return nil
}

// Environ produces a Properties instance for parsing configuration
// from environment variables with the specified prefix.
// Nested option keys are separated with underscore (_).
// Options with underscores in the name are not supported.
// For example, using the following environment variables:
//
//   test_appName=test-app
//   test_service_name=test-service
//   test_service_enabled=true
//   test_service_threshold=0.05
//   test_service_instances=10
//   test_service_true=enabled
//   test_service_10=instances
//
// with Environ(os.Environ(), "test_") call would yield the following configuration:
//
//   Map{
//     "appName": "test-app",
//     "service": Map{
//       "name": "test-service",
//       "enabled": bool(true),
//       "threshold": float64(0.05),
//       "instances": int64(10),
//       bool(true): "enabled",
//       int64(10): "instances",
//     },
//   }
//
func Environ(environ []string, prefix string) ConfigSource {
	return &KeyValueSource{
		Prefix:        prefix,
		PathSeparator: "_",
		Separator:     "=",
		Lines:         environ,
	}
}

// Arguments produces a Properties instance for parsing command-line arguments.
// Ignored configuration options may be specified with ignores vararg.
// For example, the following command-line arguments:
//
//   --appName=test-app
//   --service.name=test-service
//   --service.enabled=true OR SIMPLY --service.enabled
//   --service.threshold=0.05
//   --service.instances=10
//   --service.true=enabled
//   --service.10=instances
//
// with Arguments(os.Args[1:]) call would yield the following configuration:
//
//   Map{
//     "appName": "test-app",
//     "service": Map{
//       "name": "test-service",
//       "enabled": bool(true),
//       "threshold": float64(0.05),
//       "instances": int64(10),
//       bool(true): "enabled",
//       int64(10): "instances",
//     },
//   }
//
func Arguments(args []string, ignores ...string) ConfigSource {
	except := make(colf.Set[string])
	colf.AddAll[string](&except, colf.Slice[string](ignores))
	return &KeyValueSource{
		Prefix:        "--",
		PathSeparator: ".",
		Separator:     "=",
		Lines:         args,
		Except:        except,
	}
}

// InputSource reads configuration file paths as arguments from command line
// and parses the provided files. The codec is detected based on file extension
// (see supported extensions in ExtensionCodecs).
// The source also supports reading the configuration from stdin using the supplied codec.
// The configuration is merged from all sources, overriding options from respective sources
// in the order of appearance. This allows for building highly modular configurations with default values.
// An error (absent file, invalid format, etc.) causes a warning in logs, but does not interrupt execution.
type InputSource struct {

	// Args is a collection of command-line arguments (most commonly os.Args[1:]).
	Args []string

	// Stdin is the standard input (most commonly os.Stdin).
	Stdin flu.Input

	// FileOption is the option which will be used for supplying configuration file paths
	// (most commonly "config.file" for --config.file=config.yml style arguments).
	FileOption string

	// StdinOption is the option which will be used for supplying the stdin configuration codec
	// (most commonly "config.stdin" for --config.stdin=json style arguments).
	// If the stdin option is not specified in the argument list, the source will not read stdin.
	StdinOption string
}

// ExtensionCodecs is an index of flu.Codecs by their common file extensions.
var ExtensionCodecs = map[string]flu.Codec{
	"json": JSONViaYAML,
	"yaml": flu.YAML,
	"yml":  flu.YAML,
	"xml":  XMLViaYAML,
	"gob":  GobViaYAML,
}

// GetCodec resolves the flu.Codec using the provided extension.
func GetCodec(extension string) flu.Codec {
	extension = strings.Trim(strings.ToLower(extension), ".")
	return ExtensionCodecs[extension]
}

func (s *InputSource) ReadInto(ctx context.Context, values internal.AnyMap) error {
	fileOptionPrefix := "--" + s.FileOption + "="
	stdinOptionPrefix := "--" + s.StdinOption + "="
	for _, arg := range s.Args {
		var (
			input     flu.Input
			extension string
		)

		switch {
		case strings.HasPrefix(arg, fileOptionPrefix):
			path := arg[len(fileOptionPrefix):]
			input = flu.File(path)
			extension = strings.Trim(filepath.Ext(path), ".")
		case strings.HasPrefix(arg, stdinOptionPrefix):
			input = s.Stdin
			extension = arg[len(stdinOptionPrefix):]
		default:
			continue
		}

		codec := GetCodec(extension)
		if codec == nil {
			logf.Get(s).Warnf(ctx, "unable to find codec for extension [%s]", extension)
			continue
		}

		buf := new(flu.ByteBuffer)
		if _, err := flu.Copy(input, buf); err != nil {
			logf.Get(s).Warnf(ctx, "read %s: %s", input, err)
			continue
		}

		local := make(internal.AnyMap)
		data := flu.Bytes(os.ExpandEnv(buf.Unmask().String()))
		if err := flu.DecodeFrom(data, codec(&local)); err != nil {
			logf.Get(s).Warnf(ctx, "read expanded %s: %s", input, err)
			continue
		}

		local.SpecifyTypes()
		values.Merge(local)
	}

	return nil
}

// ConfigSources is the umbrella Parser for multiple ConfigSources.
// It merges the Map from each Parser in the order of appearance
// into one global Map and returns it as the result.
// Failures are reported as warnings and do not interrupt execution.
type ConfigSources struct {
	Name     string
	HelpText string
	Sources  []ConfigSource
}

func (cs *ConfigSources) ReadInto(ctx context.Context, values internal.AnyMap) error {
	for _, source := range cs.Sources {
		err := source.ReadInto(ctx, values)
		switch {
		case syncf.IsContextRelated(err):
			return err
		case err != nil:
			logf.Get(source).Printf(ctx, "failed to read: %v", err)
		default:
			continue
		}
	}

	return nil
}

func (cs *ConfigSources) Help() string {
	return cs.HelpText
}

func (cs *ConfigSources) String() string {
	if cs.Name != "" {
		return cs.Name
	}

	return flu.ID(cs)
}
