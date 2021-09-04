package subreddit

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	null "gopkg.in/guregu/null.v3"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"

	"github.com/jfk9w/hikkabot/core/event"
	"github.com/jfk9w/hikkabot/core/feed"
)

type CommandListener struct {
	event.Storage
	*Vendor
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

	if err := l.DeleteEvents(ctx, cmd.Chat.ID, cmd.Message.ID, cmd.User.ID, "like", "dislike"); err != nil {
		return errors.Wrap(err, "delete events")
	}

	subreddit := cmd.Args[0]
	thingID := cmd.Args[1]
	eventType := "like"
	if cmd.Key == dislikeCommandKey {
		eventType = "dislike"
	}

	log := &event.Log{
		Time:      l.Now(),
		Type:      eventType,
		ChatID:    cmd.Chat.ID,
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
		Subreddit: null.StringFrom(subreddit),
		ThingID:   null.StringFrom(thingID),
	}

	if err := l.SaveEvent(ctx, log); err != nil {
		return errors.Wrap(err, "save event")
	}

	stats, err := l.CountEvents(ctx, cmd.Chat.ID, cmd.Message.ID, "like", "dislike")
	if err != nil {
		return errors.Wrap(err, "count events")
	}

	ref := telegram.MessageRef{ChatID: cmd.Chat.ID, ID: cmd.Message.ID}
	markup := telegram.InlineKeyboard(PreferenceButtons(subreddit, thingID, stats["like"], stats["dislike"]))
	if _, err := client.EditMessageReplyMarkup(ctx, ref, markup); err != nil {
		return errors.Wrap(err, "edit reply markup")
	}

	return nil
}

func (l *CommandListener) Click(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 2 {
		return errors.Errorf("expected two arguments")
	}

	subreddit := cmd.Args[0]
	thingID := cmd.Args[1]

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

	writer := &html.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    client,
				ID:        cmd.User.ID,
				ParseMode: telegram.HTML,
			},
		},
	}

	layout := Layout{ShowAuthor: true, ShowText: true}
	writeHTML := l.writeHTML(header, layout, &things[0].Data)
	if err := writeHTML(writer); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	log := &event.Log{
		Time:      l.Now(),
		Type:      "click",
		ChatID:    cmd.Chat.ID,
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
		Subreddit: null.StringFrom(subreddit),
		ThingID:   null.StringFrom(thingID),
	}

	if err := l.SaveEvent(ctx, log); err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("save event: %s", err)
	}

	return nil
}
