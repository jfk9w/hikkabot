package main

import (
	"context"

	"github.com/jfk9w/hikkabot/internal/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/internal/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/internal/3rdparty/redditsave"
	"github.com/jfk9w/hikkabot/internal/core"
	"github.com/jfk9w/hikkabot/internal/ext/converters"
	"github.com/jfk9w/hikkabot/internal/ext/resolvers"
	"github.com/jfk9w/hikkabot/internal/ext/vendors"

	"github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type C struct {
	Telegram struct {
		tapp.Config          `yaml:",inline"`
		core.InterfaceConfig `yaml:",inline"`
	} `yaml:"telegram" doc:"Bot-related settings."`

	Db apfel.GormConfig `yaml:"db,omitempty" doc:"Poller database connection settings. Supported drivers: postgres, sqlite (not fully)" default:"{\"driver\":\"sqlite\",\"dsn\":\"file::memory:?cache=shared\"}"`

	Poller core.PollerConfig `yaml:"poller,omitempty" doc:"Poller-related settings."`

	Media struct {
		core.BlobConfig     `yaml:",inline"`
		core.MediatorConfig `yaml:",inline"`
	} `yaml:"media,omitempty" doc:"Media downloader settings."`

	FFmpeg struct {
		Enabled bool `yaml:"enabled,omitempty" doc:"Whether ffmpeg-based media converter should be enabled. Requires ffmpeg to be present in $PATH." default:"true"`
	} `yaml:"ffmpeg,omitempty" doc:"FFmpeg-related settings."`

	Aconvert struct {
		Enabled         bool `yaml:"enabled,omitempty" doc:"Whether aconvert.com-based media converter should be enabled."`
		aconvert.Config `yaml:",inline"`
	} `yaml:"aconvert,omitempty" doc:"aconvert.com-related settings."`

	Dvach dvach.Config `yaml:"dvach,omitempty" doc:"2ch.hk-related settings."`

	Reddit struct {
		Enabled       bool `yaml:"enabled,omitempty" doc:"Whether reddit.com-based vendors should be enabled."`
		reddit.Config `yaml:",inline"`
		Redditsave    redditsave.Config                  `yaml:"redditsave,omitempty" doc:"redditsave.com-related settings. Used to resolve v.redd.it videos with audio."`
		Posts         vendors.SubredditConfig            `yaml:"posts,omitempty" doc:"Subreddit posts vendor settings."`
		Suggestions   vendors.SubredditSuggestionsConfig `yaml:"suggestions,omitempty" doc:"Subreddit suggestions vendor settings."`
	} `yaml:"reddit,omitempty" doc:"reddit.com-related settings."`

	Logging    apfel.LogfConfig       `yaml:"logging,omitempty" doc:"Logging settings."`
	Prometheus apfel.PrometheusConfig `yaml:"prometheus,omitempty" doc:"Prometheus settings."`
}

func (c C) LogfConfig() apfel.LogfConfig             { return c.Logging }
func (c C) PrometheusConfig() apfel.PrometheusConfig { return c.Prometheus }
func (c C) TelegramConfig() tapp.Config              { return c.Telegram.Config }
func (c C) InterfaceConfig() core.InterfaceConfig    { return c.Telegram.InterfaceConfig }
func (c C) PollerConfig() core.PollerConfig          { return c.Poller }
func (c C) StorageConfig() apfel.GormConfig          { return c.Db }
func (c C) BlobConfig() core.BlobConfig              { return c.Media.BlobConfig }
func (c C) RedditsaveConfig() redditsave.Config      { return c.Reddit.Redditsave }
func (c C) AconvertConfig() aconvert.Config          { return c.Aconvert.Config }
func (c C) MediatorConfig() core.MediatorConfig      { return c.Media.MediatorConfig }
func (c C) DvachConfig() dvach.Config                { return c.Dvach }
func (c C) RedditConfig() reddit.Config              { return c.Reddit.Config }
func (c C) SubredditConfig() vendors.SubredditConfig { return c.Reddit.Posts }
func (c C) SubredditSuggestionsConfig() vendors.SubredditSuggestionsConfig {
	return c.Reddit.Suggestions
}

var GitCommit = "dev"

func main() {
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
				Logger: gormf.LogfLogger(app, "gorm.sql"),
			},
		}

		poller   core.Poller[C]
		telegram tapp.Mixin[C]
	)

	app.Uses(ctx,
		new(apfel.Logf[C]),
		new(apfel.Prometheus[C]),
		&telegram,
		gorm,
		&poller,
		new(core.Interface[C]),
		&resolvers.GfycatLike[C]{Name: "gfycat"},
		&resolvers.GfycatLike[C]{Name: "redgifs"},
		new(resolvers.Imgur[C]),
		new(converters.FFmpeg[C]),
		new(resolvers.Dvach[C]),
		vendors.DvachCatalog[C](),
		vendors.DvachThread[C](),
	)

	config := app.Config()

	if config.Aconvert.Enabled {
		app.Uses(ctx, new(converters.Aconvert[C]))
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
