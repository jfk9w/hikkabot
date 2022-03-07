package srstats

import (
	"context"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"hikkabot/core/aggregator"
	"hikkabot/core/feed"
	"hikkabot/ext/vendors/subreddit"
)

const (
	Name       = "srstats"
	namePrefix = Name + "/"
)

type Data struct {
	Ref         string      `json:"ref"`
	ChatID      telegram.ID `json:"chat_id"`
	FiredAtSecs int64       `json:"fired_at"`
	Options     []string    `json:"options"`
}

type Config struct {
	Enabled  bool
	Period   flu.Duration
	Interval flu.Duration
}

type Vendor struct {
	flu.Clock
	Telegram
	Events
	Feeds
	Stats
	Config
	Aliases map[string]telegram.ID
}

func (v *Vendor) Parse(ctx context.Context, ref string, options []string) (*feed.Draft, error) {
	if strings.HasPrefix(ref, namePrefix) {
		ref = ref[len(namePrefix):]
	} else {
		return nil, feed.ErrWrongVendor
	}

	var chatID telegram.ID
	if resolved, ok := v.Aliases[ref]; ok {
		chatID = resolved
	} else {
		var err error
		chatID, err = telegram.ParseID(ref)
		if err != nil {
			return nil, errors.Wrapf(err, "parse chat id: %s", ref)
		}
	}

	chat, err := v.GetChat(ctx, chatID)
	if err != nil {
		return nil, errors.Wrapf(err, "get chat %s", chatID)
	}

	return &feed.Draft{
		SubID: chatID.String(),
		Name:  chat.Title,
		Data: Data{
			Ref:     ref,
			ChatID:  chatID,
			Options: options,
		},
	}, nil
}

var (
	oneDay   = 24 * time.Hour
	twoWeeks = 2 * 14 * 24 * time.Hour
)

func (v *Vendor) Refresh(ctx context.Context, queue *feed.Queue) {
	var data Data
	if err := queue.GetData(ctx, &data); err != nil {
		_ = queue.Cancel(ctx, err)
		return
	}

	log := logrus.
		WithField("vendor", Name).
		WithField("chat_id", data.ChatID)

	now := v.Now()
	if time.Unix(data.FiredAtSecs, 0).Add(v.Interval.GetOrDefault(oneDay)).After(now) {
		return
	}

	stats, err := v.CountChatLikesBySubreddit(ctx, data.ChatID, v.Now().Add(-v.Period.GetOrDefault(twoWeeks)))
	if err != nil {
		log.Warnf("get subreddit stats: %v", err)
		return
	}

	subreddits := make(map[string]float64, len(stats))
	for sr, rating := range stats {
		subreddits[sr] = float64(rating)
	}

	suggestions, err := v.GetSuggestions(ctx, subreddits)
	if err != nil {
		log.Warnf("get subreddit suggestions: %v", err)
		return
	}

	data.FiredAtSecs = now.Unix()

	_ = queue.Proceed(ctx, func(html *html.Writer) error {
		html.Bold("suggestions").Text(" @ %s ❓", data.Ref)
		var i int
		for _, suggestion := range suggestions {
			sr := suggestion.Subreddit
			score := suggestion.Score
			header := &feed.Header{
				SubID:  sr,
				Vendor: subreddit.Name,
				FeedID: data.ChatID,
			}

			if _, err := v.Get(context.Background(), header); err == nil {
				continue
			} else if !errors.Is(err, feed.ErrNotFound) {
				log.WithField("subreddit", sr).Warnf("failed to get sub: %v", err)
				continue
			}

			html.Text("\n").
				Link(sr, "https://www.reddit.com/r/"+sr).
				Text(" – %.3f%% ", score*100).
				Link("🔥", (&telegram.Command{
					Key:  "/sub",
					Args: append([]string{"/r/" + sr, data.Ref}, data.Options...)}).
					Button("").
					StartCallbackURL(string(v.Username()))).
				Text(" ").
				Link("🛑", (&telegram.Command{
					Key:  "/sub",
					Args: append([]string{"/r/" + sr, data.Ref, aggregator.Deadborn}, data.Options...)}).
					Button("").
					StartCallbackURL(string(v.Username())))

			if i++; i >= 10 {
				break
			}
		}

		return nil
	}, data)
}
