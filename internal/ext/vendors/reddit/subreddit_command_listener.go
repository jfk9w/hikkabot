package reddit

import (
	"context"
	"net/http"

	"github.com/jfk9w/hikkabot/internal/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/internal/feed"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	like    = "like"
	dislike = "dislike"
	pre     = "pre"
	click   = "click"
)

type SubredditEventData struct {
	UserID    telegram.ID `json:"user_id" delete:"user_id" count:"-" pre:"user_id"`
	MessageID telegram.ID `json:"message_id" delete:"message_id" count:"message_id" pre:"-"`
	Subreddit string      `json:"subreddit" delete:"-" count:"-" pre:"-"`
	ThingID   string      `json:"thing_id" delete:"-" count:"-" pre:"thing_id"`
}

func newSubredditEventData(cmd *telegram.Command) SubredditEventData {
	return SubredditEventData{
		Subreddit: cmd.Args[0],
		ThingID:   cmd.Args[1],
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
	}
}

func (d SubredditEventData) filter(tagName string) (map[string]any, error) {
	result := make(map[string]any)
	config := &mapstructure.DecoderConfig{
		TagName: tagName,
		Result:  &result,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	return result, decoder.Decode(d)
}

type subredditCommandListener[C SubredditContext] struct {
	reddit   reddit.Interface
	storage  StorageInterface
	telegram telegram.Client
	writer   thingWriter[C]
}

func (l subredditCommandListener[C]) String() string {
	return "vendors.reddit.subreddit-commands"
}

func (l *subredditCommandListener[C]) CommandScope() tapp.CommandScope {
	return tapp.Public
}

func (l *subredditCommandListener[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var reddit reddit.Client[C]
	if err := app.Use(ctx, &reddit, false); err != nil {
		return err
	}

	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var telegram tapp.Mixin[C]
	if err := app.Use(ctx, &telegram, false); err != nil {
		return err
	}

	var writer thingWriter[C]
	if err := app.Use(ctx, &writer, false); err != nil {
		return err
	}

	l.reddit = reddit
	l.storage = storage
	l.telegram = telegram.Bot()
	l.writer = writer

	return nil
}

func (l *subredditCommandListener[C]) Sr_c_callback(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 3 {
		return errors.Errorf("expected two arguments")
	}

	origin, err := telegram.ParseID(cmd.Args[0])
	if err != nil {
		return err
	}

	feedID := feed.ID(origin)
	data := SubredditEventData{
		Subreddit: cmd.Args[1],
		ThingID:   cmd.Args[2],
		UserID:    cmd.User.ID,
		MessageID: cmd.Message.ID,
	}

	if origin != cmd.Chat.ID {
		// from start
		if err := l.storage.EventTx(ctx, func(tx feed.EventTx) error {
			filter, err := data.filter("pre")
			if err != nil {
				return err
			}

			return tx.GetLastEventData(feedID, pre, filter, &data)
		}); err != nil {
			return err
		}
	}

	things, err := l.reddit.GetPosts(ctx, data.Subreddit, data.ThingID)
	if err != nil {
		return errors.Wrap(err, "get post")
	}

	if len(things) == 0 {
		return errors.Wrap(err, "post not found")
	}

	html, buffer := l.createHTMLWriter(ctx)
	layout := ThingLayout{ShowAuthor: true, ShowText: true, HideMedia: true}
	writeHTML := l.writer.writeHTML(ctx, feedID, layout, things[0].Data)
	if err := writeHTML(html); err != nil {
		return errors.Wrap(err, "write html")
	}

	if err := html.Flush(); err != nil {
		return errors.Wrap(err, "flush html")
	}

	ref := telegram.MessageRef{
		ChatID: origin,
		ID:     data.MessageID,
	}

	_, err = l.telegram.CopyMessage(ctx, data.UserID, ref, &telegram.CopyOptions{
		Caption:   buffer.Pages[0],
		ParseMode: telegram.HTML,
	})

	var tgErr telegram.Error
	if errors.As(err, &tgErr) && tgErr.ErrorCode == http.StatusForbidden {
		if err := l.storage.SaveEvent(ctx, feedID, pre, data); err != nil {
			return err
		}

		err = errors.Wrap(cmd.Start(ctx, l.telegram), "send start")
	}

	if err != nil {
		return err
	}

	if err := l.storage.SaveEvent(ctx, feedID, click, data); err != nil {
		return err
	}

	_ = cmd.ReplyCallback(ctx, l.telegram, "📩")
	return nil
}

func (l *subredditCommandListener[C]) Sr_l_callback(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	if err := l.pref(ctx, cmd, like); err != nil {
		return err
	}

	_ = cmd.ReplyCallback(ctx, l.telegram, "👍")
	return nil
}

func (l *subredditCommandListener[C]) Src_dl_callback(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	if err := l.pref(ctx, cmd, dislike); err != nil {
		return err
	}

	_ = cmd.ReplyCallback(ctx, l.telegram, "👎")
	return nil
}

func (l *subredditCommandListener[C]) createHTMLWriter(ctx context.Context) (*html.Writer, *receiver.Buffer) {
	buffer := receiver.NewBuffer()
	writer := &html.Writer{Out: &output.Paged{Receiver: buffer}}
	ctx = output.With(ctx, telegram.MaxCaptionSize, 1)
	return writer.WithContext(ctx), buffer
}

func (l *subredditCommandListener[C]) pref(ctx context.Context, cmd *telegram.Command, eventType string) error {
	if len(cmd.Args) < 2 {
		return errors.Errorf("expected two arguments")
	}

	feedID := feed.ID(cmd.Chat.ID)
	data := newSubredditEventData(cmd)

	var stats map[string]int64
	if err := l.storage.EventTx(ctx, func(tx feed.EventTx) error {
		filter, err := data.filter("delete")
		if err != nil {
			return err
		}

		if err := tx.DeleteEvents(feedID, []string{like, dislike}, filter); err != nil {
			return err
		}

		if err := tx.SaveEvent(feedID, eventType, data); err != nil {
			return err
		}

		filter, err = data.filter("count")
		if err != nil {
			return err
		}

		stats, err = tx.CountEventsByType(feedID, []string{like, dislike}, filter)
		if err != nil {
			return errors.Wrap(err, "count events")
		}

		return nil
	}); err != nil {
		return err
	}

	ref := telegram.MessageRef{ChatID: cmd.Chat.ID, ID: cmd.Message.ID}

	paywall := false
	if len(cmd.Args) > 2 {
		paywall = cmd.Args[2] == "p"
	}

	var buttons []telegram.Button
	if paywall {
		buttons = []telegram.Button{PaywallButton(feedID, data.Subreddit, data.ThingID)}
	}

	buttons = append(buttons, PreferenceButtons(data.Subreddit, data.ThingID, stats[like], stats[dislike], paywall)...)
	if _, err := l.telegram.EditMessageReplyMarkup(ctx, ref, telegram.InlineKeyboard(buttons)); err != nil {
		return errors.Wrap(err, "edit reply markup")
	}

	return nil
}
