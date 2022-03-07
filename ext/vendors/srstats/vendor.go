package srstats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
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
			Ref:    ref,
			ChatID: chatID,
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
		WithField("vendor", "srstats").
		WithField("chat_id", data.ChatID)

	now := v.Now()
	if time.Unix(data.FiredAtSecs, 0).Add(v.Period.GetOrDefault(oneDay)).After(now) {
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

	totalScore := float64(0)
	for _, suggestion := range suggestions {
		totalScore += suggestion.Score
	}

	data.FiredAtSecs = now.Unix()

	_ = queue.Proceed(ctx, func(html *html.Writer) error {
		html.Text("suggestions @ %s ❓", data.Ref)
		if out, ok := html.Out.(*output.Paged); ok {
			if chat, ok := out.Receiver.(*receiver.Chat); ok {
				var buttons [][]telegram.Button
				var i int
				for _, suggestion := range suggestions {
					sr := suggestion.Subreddit
					score := suggestion.Score
					if totalScore > 0 {
						score /= totalScore
					}

					header := &feed.Header{
						SubID:  sr,
						Vendor: subreddit.Name,
						FeedID: data.ChatID,
					}

					if _, err := v.Get(context.Background(), header); !errors.Is(err, feed.ErrNotFound) {
						continue
					}

					buttons = append(buttons, []telegram.Button{
						(&telegram.Command{
							Key:  "/sub",
							Args: []string{"/r/" + sr, data.Ref},
						}).Button(fmt.Sprintf("%s [%.2f] 🔥", sr, suggestion.Score)),
						(&telegram.Command{
							Key:  "/sub",
							Args: []string{"/r/" + sr, data.Ref, aggregator.Deadborn},
						}).Button(fmt.Sprintf("%s [%.2f] 🛑", sr, suggestion.Score)),
					})

					i++
					if i >= 10 {
						break
					}
				}

				chat.ReplyMarkup = telegram.InlineKeyboard(buttons...)
			}
		}

		return nil
	}, data)
}
