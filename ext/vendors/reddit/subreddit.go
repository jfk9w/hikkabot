package reddit

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"sort"
	"sync"
	"time"

	"hikkabot/3rdparty/reddit"
	"hikkabot/core"
	"hikkabot/feed"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/colf"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"github.com/pkg/errors"
)

var subredditRegexp = regexp.MustCompile(`^(((http|https)://)?reddit\.com)?/[ur]/([0-9A-Za-z_]+)$`)

type SubredditPacingConfig struct {
	Gain    flu.Duration `yaml:"gain,omitempty" doc:"Do not apply pacing during this interval since subscription start." default:"48h"`
	Base    float64      `yaml:"base,omitempty" doc:"Base top ratio to be applied for stable subscriptions." default:"0.01"`
	Min     float64      `yaml:"min,omitempty" doc:"Lowest allowed top ratio."`
	Scale   float64      `yaml:"scale,omitempty" doc:"Top ratio multiplier. The number is highly dependent on the number of active users." default:"300"`
	Members int64        `yaml:"members,omitempty" doc:"Lowest chat members threshold." default:"300"`
	Batch   int          `yaml:"batch,omitempty" doc:"Max update batch size." default:"1"`
}

type SubredditConfig struct {
	Pacing        SubredditPacingConfig `yaml:"pacing,omitempty" doc:"Settings for controlling pacing based on top ratio."`
	CleanInterval flu.Duration          `yaml:"cleanInterval,omitempty" doc:"How often to clean things from data." default:"24h"`
	ThingTTL      flu.Duration          `yaml:"thingTtl,omitempty" doc:"How long to keep things in database." default:"168h"`
}

type SubredditContext interface {
	reddit.Context
	core.MediatorContext
	core.StorageContext
	tapp.Context
	SubredditConfig() SubredditConfig
}

type SubredditData struct {
	Subreddit     string           `json:"subreddit"`
	SentIDs       colf.Set[string] `json:"sent_ids,omitempty"`
	LastCleanSecs int64            `json:"last_clean,omitempty"`
	Layout        ThingLayout      `json:"layout,omitempty"`
}

type Subreddit[C SubredditContext] struct {
	config   SubredditConfig
	clock    syncf.Clock
	storage  StorageInterface
	mediator feed.Mediator
	client   reddit.Interface
	telegram telegram.Client
	writer   thingWriter[C]
	metrics  me3x.Registry
	mu       sync.RWMutex
}

func (v *Subreddit[C]) String() string {
	return "subreddit"
}

func (v *Subreddit[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var mediator core.Mediator[C]
	if err := app.Use(ctx, &mediator, false); err != nil {
		return err
	}

	var client reddit.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	var bot tapp.Mixin[C]
	if err := app.Use(ctx, &bot, false); err != nil {
		return err
	}

	var writer thingWriter[C]
	if err := app.Use(ctx, &writer, false); err != nil {
		return err
	}

	var listener subredditCommandListener[C]
	if err := app.Use(ctx, &listener, false); err != nil {
		return err
	}

	var metrics apfel.Prometheus[C]
	if err := app.Use(ctx, &metrics, false); err != nil {
		return err
	}

	v.config = app.Config().SubredditConfig()
	v.clock = app
	v.storage = storage
	v.client = client
	v.telegram = bot.Bot()
	v.writer = writer
	v.metrics = metrics.Registry().WithPrefix("app_subreddit")

	return nil
}

func (v *Subreddit[C]) BeforeResume(ctx context.Context, header feed.Header) error {
	return v.client.Subscribe(ctx, reddit.Subscribe, []string{header.SubID})
}

func (v *Subreddit[C]) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	groups := subredditRegexp.FindStringSubmatch(ref)
	if len(groups) != 5 {
		return nil, nil
	}

	subreddit := groups[4]
	things, err := v.getListing(ctx, subreddit, 1)
	if err != nil {
		return nil, errors.Wrap(err, "get listing")
	}

	if len(things) > 0 {
		subreddit = things[0].Data.Subreddit
	}

	data := &SubredditData{Subreddit: subreddit}
	for _, option := range options {
		switch option {
		case "t":
			data.Layout.ShowText = true
		case "!m":
			data.Layout.HideMedia = true
		case "u":
			data.Layout.ShowAuthor = true
		case "p":
			data.Layout.ShowPaywall = true
			data.Layout.HideMediaLink = true
			data.Layout.HideLink = true
			data.Layout.HideTitle = true
		case "l":
			data.Layout.ShowPreference = true
		}
	}

	return &feed.Draft{
		SubID: subreddit,
		Name:  getSubredditName(subreddit),
		Data:  data,
	}, nil
}

func (v *Subreddit[C]) Refresh(ctx context.Context, header feed.Header, refresh feed.Refresh) error {
	var data SubredditData
	if err := refresh.Init(ctx, &data); err != nil {
		return err
	}

	things, err := v.getListing(ctx, data.Subreddit, 100)
	if err != nil {
		if errors.As(err, new(net.Error)) {
			return nil
		} else if errors.As(err, new(*json.SyntaxError)) {
			return nil
		} else if codeErr := new(httpf.StatusCodeError); errors.As(err, codeErr) &&
			(codeErr.StatusCode < 400 || codeErr.StatusCode >= 500) {
			return nil
		}

		return err
	}

	if err := v.storage.SaveThings(ctx, things); err != nil {
		return err
	}

	var (
		count      = 0
		cleanData  = syncf.Lazy[any](func(ctx context.Context) (any, error) { return nil, v.cleanData(ctx, &data) })
		percentile = syncf.Lazy[int](func(ctx context.Context) (int, error) { return v.getPercentile(ctx, header, data) })
	)

	for _, thing := range things {
		thing := thing.Data
		if data.SentIDs[thing.ID] {
			continue
		}

		percentile, err := percentile.Get(ctx)
		if err != nil {
			return err
		}

		if thing.Ups < percentile || thing.IsSelf && !data.Layout.ShowText || !thing.IsSelf && data.Layout.HideMedia {
			continue
		}

		writeHTML := v.writer.writeHTML(ctx, header.FeedID, data.Layout, thing)
		if writeHTML == nil {
			continue
		}

		if _, err := cleanData.Get(ctx); err != nil {
			return err
		}

		data.SentIDs.Add(thing.ID)
		if err := refresh.Submit(ctx, writeHTML, data); err != nil {
			return err
		}

		count++
		if count >= v.config.Pacing.Batch {
			break
		}
	}

	return nil
}

func (v *Subreddit[C]) cleanData(ctx context.Context, data *SubredditData) error {
	now := v.clock.Now()
	if now.Sub(time.Unix(data.LastCleanSecs, 0)) < v.config.CleanInterval.Value {
		return nil
	}

	return v.storage.RedditTx(ctx, func(tx StorageTx) error {
		deletedThings, err := tx.DeleteStaleThings(now.Add(-v.config.ThingTTL.Value))
		if err != nil {
			return err
		}

		if deletedThings > 0 {
			logf.Get(v).Infof(ctx, "deleted %d stale things", deletedThings)
		}

		freshIDs, err := tx.GetFreshThingIDs(data.SentIDs)
		if err != nil {
			return err
		}

		data.SentIDs = freshIDs
		data.LastCleanSecs = now.Unix()

		return nil
	})
}

func (v *Subreddit[C]) getPercentile(ctx context.Context, header feed.Header, data SubredditData) (int, error) {
	members, err := v.telegram.GetChatMemberCount(ctx, telegram.ID(header.FeedID))
	if err != nil {
		return 0, err
	}

	v.metrics.Gauge("subscribers", me3x.Labels{}.Add("feed_id", header.FeedID)).Set(float64(members))

	pacing := v.config.Pacing
	var percentile int
	return percentile, v.storage.RedditTx(ctx, func(tx StorageTx) error {
		boost := 0.
		if (data.Layout.ShowPreference || data.Layout.ShowPaywall) && len(data.SentIDs) > 0 {
			score, err := tx.Score(header.FeedID, colf.ToSlice[string](data.SentIDs))
			switch {
			case err == nil && score.First != nil && v.clock.Now().Sub(*score.First) >= pacing.Gain.Value:
				thingRatio := (float64(score.LikedThings) - float64(score.DislikedThings)) / float64(len(data.SentIDs))
				if members < pacing.Members {
					members = pacing.Members
				}

				userRatio := (float64(score.Likes) - float64(score.Dislikes)) / float64(members)
				boost = pacing.Scale * thingRatio * userRatio
			case err != nil:
				logf.Get(v).Warnf(ctx, "using base pacing %.4f due to score error: %v", pacing.Base, err)
			}
		}

		top := pacing.Base * (boost + 1)
		if top < pacing.Min {
			top = pacing.Min
		}

		v.metrics.Gauge("top", header.Labels()).Set(top)

		var err error
		percentile, err = tx.GetPercentile(data.Subreddit, top)
		if err != nil {
			return errors.Wrap(err, "get percentile")
		}

		return nil
	})
}

func (v *Subreddit[C]) getListing(ctx context.Context, subreddit string, limit int) ([]reddit.Thing, error) {
	things, err := v.client.GetListing(ctx, subreddit, "hot", limit)
	if err != nil {
		return nil, err
	}

	now := v.clock.Now()
	for i := range things {
		things[i].LastSeen = now
	}

	sort.Sort(thingSorter(things))
	return things, nil
}

func getSubredditName(subreddit string) string {
	return "#" + subreddit
}
