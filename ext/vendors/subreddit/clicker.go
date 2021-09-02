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

type Clicker struct {
	event.Storage
	*Vendor
}

func (v *Clicker) OnCommand(ctx context.Context, client telegram.Client, cmd *telegram.Command) (bool, error) {
	switch cmd.Key {
	case clickCommandKey:
		err := v.Click(ctx, client, cmd)
		if err != nil {
			if tgerr := new(telegram.Error); errors.As(err, tgerr) && tgerr.ErrorCode == http.StatusForbidden {
				err = cmd.Start(ctx, client)
			}
		}

		return true, err
	}

	return false, nil
}

func (v *Clicker) Click(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 2 {
		return errors.Errorf("expected two arguments")
	}

	subreddit := cmd.Args[0]
	thingID := cmd.Args[1]

	header := &feed.Header{
		SubID:  subreddit,
		Vendor: "tracker",
		FeedID: cmd.User.ID,
	}

	data := &Data{
		MediaOnly:   false,
		IndexUsers:  true,
		TrackClicks: false,
	}

	things, err := v.RedditClient.GetPosts(ctx, subreddit, thingID)
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

	writeHTML := v.writeHTML(header, data, &things[0].Data)
	if err := writeHTML(writer); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	log := &event.Log{
		Time:      v.Now(),
		Type:      "click",
		ChatID:    cmd.Chat.ID,
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
		Subreddit: null.StringFrom(subreddit),
		ThingID:   null.StringFrom(thingID),
	}

	if err := v.SaveEvent(ctx, log); err != nil {
		logrus.WithFields(cmd.Labels().Map()).Warnf("save event: %s", err)
	}

	return nil
}
