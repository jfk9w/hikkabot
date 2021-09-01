package subreddit

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

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

type Tracked struct {
	event.Storage
	*Vendor
}

func (t *Tracked) OnCommand(ctx context.Context, client telegram.Client, cmd *telegram.Command) (bool, error) {
	switch cmd.Key {
	case clickCommandKey:
		return true, t.Track(ctx, client, cmd)
	case "/start":
		payload, err := base64.URLEncoding.DecodeString(cmd.Payload)
		if err != nil {
			return false, nil
		}

		tokens := strings.Split(string(payload), ",")
		if len(tokens) != 3 && tokens[0] != "sr" {
			return false, nil
		}

		subreddit := tokens[1]
		thingID := tokens[2]

		t.saveEvent(ctx, cmd, subreddit, thingID)
		return true, t.sendPost(ctx, client, subreddit, thingID, cmd.User.ID)
	}

	return false, nil
}

func (t *Tracked) Track(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) != 2 {
		return errors.Errorf("expected two arguments")
	}

	subreddit := cmd.Args[0]
	thingID := cmd.Args[1]

	if ok, err := t.IsKnownUser(ctx, cmd.User.ID); err != nil {
		return errors.Wrap(err, "internal error")
	} else if ok {
		t.saveEvent(ctx, cmd, subreddit, thingID)
		return t.sendPost(ctx, client, subreddit, thingID, cmd.User.ID)
	}

	payload := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("sr,%s,%s", subreddit, thingID)))
	url := fmt.Sprintf("https://t.me/%s?start=%s", client.Username(), payload)
	if ok, err := client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &telegram.AnswerOptions{URL: url}); err != nil {
		return errors.Wrap(err, "answer")
	} else if !ok {
		return errors.New("invalid answer")
	}

	return nil
}

func (t *Tracked) saveEvent(ctx context.Context, cmd *telegram.Command, subreddit, thingID string) {
	log := &event.Log{
		Time:      t.Now(),
		Type:      "click",
		ChatID:    cmd.Chat.ID,
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
		Subreddit: null.StringFrom(subreddit),
		ThingID:   null.StringFrom(thingID),
	}

	if err := t.SaveEvent(ctx, log); err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("save event: %s", err)
	}
}

func (t *Tracked) sendPost(ctx context.Context, client telegram.Client, subreddit, thingID string, userID telegram.ID) error {
	header := &feed.Header{
		SubID:  subreddit,
		Vendor: "tracker",
		FeedID: userID,
	}

	data := &Data{
		MediaOnly:   false,
		IndexUsers:  true,
		TrackClicks: false,
	}

	things, err := t.RedditClient.GetPosts(ctx, subreddit, thingID)
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
				ID:        userID,
				ParseMode: telegram.HTML,
			},
		},
	}

	writeHTML, err := t.writeHTML(header, data, &things[0].Data)
	if err != nil {
		return err
	}

	if err := writeHTML(writer); err != nil {
		return errors.Wrap(err, "write HTML")
	}

	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "flush HTML")
	}

	return nil
}
