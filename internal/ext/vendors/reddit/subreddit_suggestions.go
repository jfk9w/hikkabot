package reddit

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jfk9w/hikkabot/v4/internal/3rdparty/srstats"
	"github.com/jfk9w/hikkabot/v4/internal/core"
	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"github.com/pkg/errors"
)

type SubredditSuggestionsConfig struct {
	Period   flu.Duration `yaml:"period,omitempty" doc:"Period to consider data for." default:"374h"`
	Interval flu.Duration `yaml:"interval,omitempty" doc:"How often to make suggestions." default:"24h"`
}

type SubredditSuggestionsContext interface {
	tapp.Context
	core.StorageContext
	core.InterfaceContext
	SubredditSuggestionsConfig() SubredditSuggestionsConfig
}

type SubredditSuggestionsData struct {
	Ref         string   `json:"ref"`
	FeedID      feed.ID  `json:"chat_id"`
	FiredAtSecs int64    `json:"fired_at"`
	Options     []string `json:"options"`
}

type SubredditSuggestions[C SubredditSuggestionsContext] struct {
	clock    syncf.Clock
	config   SubredditSuggestionsConfig
	telegram telegram.Client
	storage  StorageInterface
	client   srstats.Client[C]
	aliases  map[string]telegram.ID
}

func (v SubredditSuggestions[C]) String() string {
	return "srstats"
}

func (v *SubredditSuggestions[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var bot tapp.Mixin[C]
	if err := app.Use(ctx, &bot, false); err != nil {
		return err
	}

	var client srstats.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	v.clock = app
	v.config = app.Config().SubredditSuggestionsConfig()
	v.telegram = bot.Bot()
	v.storage = storage
	v.client = client
	v.aliases = app.Config().InterfaceConfig().Aliases

	return nil
}

func (v *SubredditSuggestions[C]) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	if !strings.HasPrefix(ref, v.String()+"/") {
		return nil, nil

	}

	ref = ref[len(v.String())+1:]

	var chatID telegram.ID
	if resolved, ok := v.aliases[ref]; ok {
		chatID = resolved
	} else {
		var err error
		chatID, err = telegram.ParseID(ref)
		if err != nil {
			return nil, errors.Wrapf(err, "parse chat id: %s", ref)
		}
	}

	chat, err := v.telegram.GetChat(ctx, chatID)
	if err != nil {
		return nil, errors.Wrapf(err, "get chat %s", chatID)
	}

	return &feed.Draft{
		SubID: chatID.String(),
		Name:  chat.Title,
		Data: SubredditSuggestionsData{
			Ref:     ref,
			FeedID:  feed.ID(chatID),
			Options: options,
		},
	}, nil
}

func (v *SubredditSuggestions[C]) Refresh(ctx context.Context, header feed.Header, refresh feed.Refresh) error {
	var data SubredditSuggestionsData
	if err := refresh.Init(ctx, &data); err != nil {
		return err
	}

	now := v.clock.Now()
	if time.Unix(data.FiredAtSecs, 0).Add(v.config.Interval.Value).After(now) {
		return nil
	}

	multipliers := map[string]float64{
		"like":    1,
		"click":   1,
		"dislike": -1,
	}

	since := v.clock.Now().Add(-v.config.Period.Value)
	stats, err := v.storage.CountEventsBy(ctx, data.FeedID, since, "subreddit", multipliers)
	if err != nil {
		return err
	}

	subreddits := make(map[string]float64, len(stats))
	for sr, rating := range stats {
		subreddits[sr] = float64(rating)
	}

	suggestions, err := v.getSuggestions(ctx, subreddits)
	if err != nil {
		return err
	}

	data.FiredAtSecs = now.Unix()
	writeHTML := v.writeHTML(data, suggestions)
	return refresh.Submit(ctx, writeHTML, data)
}

func (v *SubredditSuggestions[C]) writeHTML(data SubredditSuggestionsData, suggestions suggestions) feed.WriteHTML {
	return func(html *html.Writer) error {
		html.Bold("suggestions").Text(" @ %s ❓", data.Ref)
		var i int
		for _, suggestion := range suggestions {
			sr := suggestion.subreddit
			score := suggestion.score
			header := feed.Header{
				SubID:  sr,
				Vendor: "subreddit",
				FeedID: data.FeedID,
			}

			if _, err := v.storage.GetSubscription(context.Background(), header); !errors.Is(err, feed.ErrNotFound) {
				continue
			}

			html.Text("\n").
				Link(sr, "https://www.reddit.com/r/"+sr).
				Text(" – %.3f%% ", score*100).
				Link("🔥", (&telegram.Command{
					Key:  "sub",
					Args: append([]string{"/r/" + sr, data.Ref}, data.Options...)}).
					Button("").
					StartCallbackURL(string(v.telegram.Username()))).
				Text(" ").
				Link("🛑", (&telegram.Command{
					Key:  "sub",
					Args: append([]string{"/r/" + sr, data.Ref, feed.Deadborn}, data.Options...)}).
					Button("").
					StartCallbackURL(string(v.telegram.Username())))

			if i++; i >= 10 {
				break
			}
		}

		return nil
	}
}

func (v *SubredditSuggestions[C]) getSuggestions(ctx context.Context, subreddits map[string]float64) (suggestions, error) {
	global, err := v.client.GetGlobalHistogram(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get global histogram")
	}

	normalize(global)
	m := make(map[string]float64)
	for subreddit, multiplier := range subreddits {
		histogram, err := v.client.GetHistogram(ctx, subreddit)
		if err != nil {
			logf.Get(v).Warnf(ctx, "failed to get histogram for [%s]: %v", subreddit, err)
			continue
		}

		normalize(histogram)
		for subreddit, score := range histogram {
			globalScore := global[subreddit]
			if globalScore < 0.0001 {
				continue
			}

			m[subreddit] += multiplier * score / globalScore
		}
	}

	normalize(m)
	suggestions := make(suggestions, len(m))
	ids := make([]string, len(m))
	i := 0
	for subreddit, score := range m {
		suggestions[i] = suggestion{subreddit: subreddit, score: score}
		ids[i] = subreddit
		i++
	}

	names, err := v.client.GetSubredditNames(ctx, ids)
	if err != nil {
		return nil, errors.Wrap(err, "get subreddit names")
	}

	for i := range suggestions {
		if len(names) <= i {
			suggestions = suggestions[:len(names)]
			break
		}

		suggestions[i].subreddit = names[i]
	}

	sort.Sort(suggestions)
	return suggestions, nil
}

type suggestion struct {
	subreddit string
	score     float64
}

type suggestions []suggestion

func (s suggestions) Len() int {
	return len(s)
}

func (s suggestions) Less(i, j int) bool {
	return s[i].score > s[j].score
}

func (s suggestions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func normalize(histogram map[string]float64) {
	sum := 0.
	for _, value := range histogram {
		sum += value
	}

	if sum > 0 {
		for key, value := range histogram {
			histogram[key] = value / sum
		}
	}
}
