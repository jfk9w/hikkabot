package main

import (
	"context"

	"hikkabot/3rdparty/dvach"
	"hikkabot/3rdparty/reddit"
	"hikkabot/3rdparty/viddit"
	"hikkabot/core"
	"hikkabot/ext/converters"
	"hikkabot/ext/resolvers"
	"hikkabot/ext/vendors"

	"github.com/jfk9w-go/flu/gormf"
	"gorm.io/gorm"

	"gorm.io/driver/sqlite"

	"github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"gorm.io/driver/postgres"
)

type C struct {
	Telegram struct {
		tapp.Config          `yaml:",inline"`
		core.InterfaceConfig `yaml:",inline"`
	} `yaml:"telegram" doc:"Bot-related settings."`

	Poller struct {
		core.PollerConfig `yaml:",inline"`
		Db                apfel.GormConfig `yaml:"db" doc:"Poller database connection settings. Supported drivers: postgres, sqlite"`
		Media             struct {
			core.BlobConfig     `yaml:",inline"`
			core.MediatorConfig `yaml:",inline"`
		} `yaml:"media,omitempty" doc:"Media downloader settings."`
	} `yaml:"poller" doc:"Poller-related settings."`

	Viddit viddit.Config `yaml:"viddit,omitempty" doc:"viddit.com-related settings."`

	Aconvert struct {
		Enabled         bool `yaml:"enabled,omitempty" doc:"Whether aconvert.com-based converter should be enabled."`
		aconvert.Config `yaml:",inline"`
	} `yaml:"aconvert,omitempty" doc:"aconvert.com-related settings."`

	FFmpeg struct {
		Enabled bool `yaml:"enabled,omitempty" doc:"Whether ffmpeg-based converter should be enabled."`
	} `yaml:"ffmpeg,omitempty" doc:"ffmpeg-related settings."`

	Dvach struct {
		Enabled      bool `yaml:"enabled,omitempty" doc:"Whether 2ch.hk-based vendors should be enabled."`
		dvach.Config `yaml:",inline"`
	} `yaml:"dvach,omitempty" doc:"2ch.hk-related settings."`

	Reddit struct {
		Enabled       bool `yaml:"enabled,omitempty" doc:"Whether reddit.com-based vendors should be enabled."`
		reddit.Config `yaml:",inline"`
		Posts         vendors.SubredditConfig            `yaml:"posts,omitempty" doc:"Subreddit posts settings."`
		Suggestions   vendors.SubredditSuggestionsConfig `yaml:"suggestions,omitempty" doc:"Subreddit suggestions settings."`
	} `yaml:"reddit,omitempty" doc:"reddit.com-related settings."`

	Logging    apfel.LogfConfig       `yaml:"logging,omitempty" doc:"Logging settings."`
	Prometheus apfel.PrometheusConfig `yaml:"prometheus,omitempty" doc:"Prometheus settings."`
}

func (c C) LogfConfig() apfel.LogfConfig             { return c.Logging }
func (c C) PrometheusConfig() apfel.PrometheusConfig { return c.Prometheus }
func (c C) TelegramConfig() tapp.Config              { return c.Telegram.Config }
func (c C) InterfaceConfig() core.InterfaceConfig    { return c.Telegram.InterfaceConfig }
func (c C) PollerConfig() core.PollerConfig          { return c.Poller.PollerConfig }
func (c C) StorageConfig() apfel.GormConfig          { return c.Poller.Db }
func (c C) BlobConfig() core.BlobConfig              { return c.Poller.Media.BlobConfig }
func (c C) VidditConfig() viddit.Config              { return c.Viddit }
func (c C) AconvertConfig() aconvert.Config          { return c.Aconvert.Config }
func (c C) MediatorConfig() core.MediatorConfig      { return c.Poller.Media.MediatorConfig }
func (c C) DvachConfig() dvach.Config                { return c.Dvach.Config }
func (c C) RedditConfig() reddit.Config              { return c.Reddit.Config }
func (c C) SubredditConfig() vendors.SubredditConfig { return c.Reddit.Posts }
func (c C) SubredditSuggestionsConfig() vendors.SubredditSuggestionsConfig {
	return c.Reddit.Suggestions
}

var GitCommit = "dev"

func main() {
	logf.ResetLevel(logf.Trace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := apfel.Boot[C]{
		Name:    "hikkabot",
		Version: GitCommit,
	}.App(ctx)
	defer flu.CloseQuietly(app)

	var (
		gorm = &apfel.Gorm[C]{
			Drivers: map[string]apfel.GormDriver{
				"postgres": postgres.Open,
				"sqlite":   sqlite.Open,
			},
			Config: gorm.Config{
				Logger: gormf.LogfLogger(app, func() logf.Interface { return logf.Get("gorm.sql") }),
			},
		}

		poller   core.Poller[C]
		telegram tapp.Mixin[C]
	)

	app.Uses(ctx,
		new(apfel.Logf[C]),
		new(apfel.Metrics[C]),
		&telegram,
		gorm,
		&poller,
		new(core.Interface[C]),
		&resolvers.GfycatLike[C]{Name: "gfycat"},
		&resolvers.GfycatLike[C]{Name: "redgifs"},
		new(resolvers.Imgur[C]),
	)

	config := app.Config()

	if config.FFmpeg.Enabled {
		app.Uses(ctx, new(converters.FFmpeg[C]))
	}

	if config.Aconvert.Enabled {
		app.Uses(ctx, new(converters.Aconvert[C]))
	}

	if config.Dvach.Enabled {
		app.Uses(ctx,
			new(resolvers.Dvach[C]),
			vendors.DvachCatalog[C](),
			vendors.DvachThread[C](),
		)
	}

	if config.Reddit.Enabled {
		app.Uses(ctx,
			new(resolvers.Reddit[C]),
			vendors.Subreddit[C](),
			vendors.SubredditSuggestions[C](),
		)
	}

	if err := poller.RestoreActive(ctx); err != nil {
		logf.Panicf(ctx, "restore active: %+v", err)
	}

	telegram.Run(ctx)
}
