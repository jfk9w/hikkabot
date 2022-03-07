package subreddit

import (
	"context"
	"net/http"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"
	"gorm.io/gorm"

	"hikkabot/core/event"
	"hikkabot/core/feed"
)

type CommandListener struct {
	event.Storage
	*Vendor
	SupervisorID telegram.ID
}

func (l *CommandListener) OnCommand(ctx context.Context, client telegram.Client, cmd *telegram.Command) (bool, error) {
	switch cmd.Key {
	case clickCommandKey:
		err := l.Click(ctx, client, cmd)
		if err != nil {
			if tgerr := new(telegram.Error); errors.As(err, tgerr) && tgerr.ErrorCode == http.StatusForbidden {
				err = cmd.Start(ctx, client)
			}
		}

		return true, err
	case likeCommandKey, dislikeCommandKey:
		return true, l.Pref(ctx, client, cmd)
	}

	return false, nil
}

func (l *CommandListener) Pref(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 2 {
		return errors.Errorf("expected two arguments")
	}

	var stats map[string]int64
	update := func(storage event.Storage) error {
		if err := storage.DeleteEvents(ctx, cmd.Chat.ID, cmd.Message.ID, cmd.User.ID, "like", "dislike"); err != nil {
			return errors.Wrap(err, "delete events")
		}

		if err := storage.SaveEvent(ctx, l.newEvent(cmd)); err != nil {
			return errors.Wrap(err, "save event")
		}

		var err error
		stats, err = storage.CountEvents(ctx, cmd.Chat.ID, cmd.Message.ID, "like", "dislike")
		if err != nil {
			return errors.Wrap(err, "count events")
		}

		return nil
	}

	if storage, ok := l.Storage.(*event.SQLStorage); ok {
		if err := storage.Unmask().WithContext(ctx).
			Transaction(func(tx *gorm.DB) error { return update((*event.SQLStorage)(tx)) }); err != nil {
			return err
		}
	} else if err := update(l.Storage); err != nil {
		return err
	}

	ref := telegram.MessageRef{ChatID: cmd.Chat.ID, ID: cmd.Message.ID}
	subreddit, thingID := cmd.Args[0], cmd.Args[1]
	paywall := false
	if len(cmd.Args) > 2 {
		paywall = cmd.Args[2] == "p"
	}

	var buttons []telegram.Button
	if paywall {
		buttons = []telegram.Button{PaywallButton(subreddit, thingID)}
	}

	buttons = append(buttons, PreferenceButtons(subreddit, thingID, stats["like"], stats["dislike"], paywall)...)
	if _, err := client.EditMessageReplyMarkup(ctx, ref, telegram.InlineKeyboard(buttons)); err != nil {
		return errors.Wrap(err, "edit reply markup")
	}

	return nil
}

func (l *CommandListener) Click(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 2 {
		return errors.Errorf("expected two arguments")
	}

	subreddit, thingID := cmd.Args[0], cmd.Args[1]
	header := &feed.Header{
		SubID:  subreddit,
		Vendor: "click_tracker",
		FeedID: cmd.User.ID,
	}

	things, err := l.RedditClient.GetPosts(ctx, subreddit, thingID)
	if err != nil {
		return errors.Wrap(err, "get post")
	}

	if len(things) == 0 {
		return errors.Wrap(err, "post not found")
	}

	buffer := receiver.NewBuffer()
	writer := &html.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver:  buffer,
			PageCount: 1,
			PageSize:  telegram.MaxCaptionSize,
		},
	}

	layout := Layout{ShowAuthor: true, ShowText: true, HideMedia: true}
	writeHTML := l.writeHTML(header, layout, &things[0].Data)
	if err := writeHTML(writer); err != nil {
		return errors.Wrap(err, "write html")
	}

	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "flush html")
	}

	ref := telegram.MessageRef{
		ChatID: cmd.Chat.ID,
		ID:     cmd.Message.ID,
	}

	if _, err := client.CopyMessage(ctx, cmd.User.ID, ref, &telegram.CopyOptions{
		Caption:   buffer.Pages[0],
		ParseMode: telegram.HTML,
	}); err != nil {
		return errors.Wrap(err, "copy message")
	}

	if err := l.SaveEvent(ctx, l.newEvent(cmd)); err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("save event: %s", err)
	}

	return nil
}

func (l *CommandListener) newEvent(cmd *telegram.Command) *event.Log {
	var eventType string
	switch cmd.Key {
	case clickCommandKey:
		eventType = "click"
	case likeCommandKey:
		eventType = "like"
	case dislikeCommandKey:
		eventType = "dislike"
	}

	return &event.Log{
		Time:      l.Now(),
		Type:      eventType,
		ChatID:    cmd.Chat.ID,
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
		Subreddit: null.StringFrom(cmd.Args[0]),
		ThingID:   null.StringFrom(cmd.Args[1]),
	}
}
