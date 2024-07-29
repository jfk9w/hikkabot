package apfel

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel/internal"
	"github.com/jfk9w-go/flu/apfel/schema"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

var BootHelpTemplate = `
  Available options:

    --help                  – print this message
    --version               – print application name and version
    --config.schema=<codec> – print configuration schema to stdout
    --config.values=<codec> – print configuration values to stdout

  If you want to generate configuration schema for use in IDEA or other IDEs: 

    ./%s --config.schema=json > config.schema.yaml

  If you want to generate configuration template for further customization:

    ./%s --config.values=yml > config.yml
`

// Help may be implemented by a ConfigSource in order to injection
// help message into `--help` output.
type Help interface {
	Help() string
}

// Boot implements common CLI interface (see BootHelpTemplate).
// It also helps initialize Core instance.
type Boot[C any] struct {

	// Name is the application name.
	// It is also used as environment variable prefix for reading configuration.
	Name string

	// Version is the application version.
	// Generally you do not want to statically set it,
	// but instead define a variable and set via ldflags during build.
	Version string

	// Desc is the optional application description.
	// It will be printed along with BootHelpTemplate when using `--help` CLI option.
	Desc string

	// Clock is the optional clock which should be used for the application instance.
	// If not set, syncf.DefaultClock will be used (basically time.Now()).
	Clock syncf.Clock

	// Stdout is the optional application stdout.
	// If not set, os.Stdout will be used.
	Stdout flu.Output

	// Source is the optional ConfigSource for the application instance.
	// If not set, DefaultConfigSource will be used.
	Source ConfigSource

	// Quit is the optional exit function for calling when common CLI options are processed.
	// By default, os.Exit(0) is used.
	Quit func()
}

func (b Boot[C]) withDefaults() Boot[C] {
	if b.Stdout == nil {
		b.Stdout = flu.IO{W: os.Stdout}
	}

	if b.Clock == nil {
		b.Clock = syncf.DefaultClock
	}

	if b.Quit == nil {
		b.Quit = func() { os.Exit(0) }
	}

	if b.Source == nil && b.Name != "" {
		b.Source = DefaultConfigSource(b.Name)
	}

	if b.Version == "" {
		b.Version = "dev"
	}

	return b
}

// App reads configuration properties, processes common "immediate" CLI arguments (see BootHelpTemplate)
// and sets up a Core instance. Application should not set up anything prior to calling App method,
// since it may call os.Exit.
//
// Configuration schema is resolved automatically from C structure fields and their tags.
// You can see LogfConfig or GormConfig for examples on how to define configuration structs and annotate fields.
// Note that `yaml` tags are always used for marshalling/unmarshalling no matter what the codec is.
// Also note that schema resolution has some limitations and sometimes a little help is needed.
// For example, it is recommended to use `example` and/or `default` tags as often as possible, since
// the framework sometimes fail to resolve JSON type by its Go type (and rightfully so – this is impossible,
// especially when doing some exotic stuff like marshalling flu.Set into JSON array).
// Some limitations when filling default values may also apply
// (you can use `--config.values` and `--config.schema` for debug).
//
// See package schema for more details on schema resolution.
//goland:noinspection GoAssignmentToReceiver
func (b Boot[C]) App(ctx context.Context) *Core[C] {
	b = b.withDefaults()
	id := rootLoggerName + "." + flu.Readable(b.Source)
	values := make(internal.AnyMap)
	if err := b.Source.ReadInto(ctx, values); err != nil {
		logf.Get(id).Panicf(ctx, "read config values: %v", err)
	}

	var printVersion bool
	if err := values.As(&printVersion, "version"); err != nil && !errors.Is(err, internal.ErrKeyNotFound) {
		logf.Get(id).Panicf(ctx, "parse version parameter: %v", err)
	}

	if printVersion {
		text := flu.Bytes(fmt.Sprintf("%s %s", b.Name, b.Version))
		if _, err := flu.Copy(text, b.Stdout); err != nil {
			logf.Get(id).Panicf(ctx, "print name and version: %v", err)
		}

		b.Quit()
	}

	var printHelp bool
	if err := values.As(&printHelp, "help"); err != nil && !errors.Is(err, internal.ErrKeyNotFound) {
		logf.Get(id).Panicf(ctx, "parse help parameter: %v", err)
	}

	if printHelp {
		helpText := b.Desc + fmt.Sprintf(BootHelpTemplate, b.Name, b.Name)
		if help, ok := b.Source.(Help); ok {
			helpText += help.Help()
		}

		if _, err := flu.Copy(flu.Bytes(helpText), b.Stdout); err != nil {
			logf.Get(id).Panicf(ctx, "print help: %v", err)
		}

		b.Quit()
	}

	var config C
	schema, err := schema.Generate(reflect.TypeOf(config))
	if err != nil {
		logf.Get(id).Panicf(ctx, "generate config schema: %+v", err)
	}

	var printSchema string
	if err := values.As(&printSchema, "config", "schema"); err != nil && !errors.Is(err, internal.ErrKeyNotFound) {
		logf.Get(id).Panicf(ctx, "parse config.schema parameter: %v", err)
	}

	if codec, ok := ExtensionCodecs[printSchema]; ok {
		if err := flu.EncodeTo(codec(schema), b.Stdout); err != nil {
			logf.Get(id).Panicf(ctx, "write config schema: %v", err)
		}

		b.Quit()
	}

	if err := schema.ApplyDefaults(&config, false); err != nil {
		logf.Get(id).Panicf(ctx, "apply pre config defaults: %v", err)
	}

	if err := values.As(&config); err != nil {
		logf.Get(id).Panicf(ctx, "transform config values: %v", err)
	}

	if err := schema.ApplyDefaults(&config, true); err != nil {
		logf.Get(id).Panicf(ctx, "apply post config defaults: %v", err)
	}

	var printValues string
	if err := values.As(&printValues, "config", "values"); err != nil && !errors.Is(err, internal.ErrKeyNotFound) {
		logf.Get(id).Panicf(ctx, "parse config.values parameter: %v", err)
	}

	if codec, ok := ExtensionCodecs[printValues]; ok {
		if err := flu.EncodeTo(codec(config), b.Stdout); err != nil {
			logf.Get(id).Panicf(ctx, "write config values: %v", err)
		}

		b.Quit()
	}

	return &Core[C]{
		Clock:   b.Clock,
		config:  config,
		id:      id,
		version: b.Version,
	}
}
