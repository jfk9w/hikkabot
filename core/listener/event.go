package listener

import (
	"context"
	"fmt"

	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/sirupsen/logrus"

	"hikkabot/core/feed"
)

type writeHTMLWithChatLink func(html *html.Writer, chatLink string) *html.Writer

type Event struct {
	AccessControl
	me3x.Registry
}

func (l *Event) OnResume(ctx context.Context, client telegram.Client, sub *feed.Subscription) error {
	buttons := []telegram.Button{
		(&telegram.Command{Key: suspendCommandKey, Args: []string{sub.Header.String()}}).Button("Suspend"),
	}

	html := func(html *html.Writer, chatLink string) *html.Writer {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🔥")
	}

	l.Counter("resume", sub.Labels()).Inc()
	return l.notify(ctx, client, sub.FeedID, telegram.InlineKeyboard(buttons), html)
}

func (l *Event) OnSuspend(ctx context.Context, client telegram.Client, sub *feed.Subscription) error {
	buttons := []telegram.Button{
		(&telegram.Command{Key: resumeCommandKey, Args: []string{sub.Header.String()}}).Button("Resume"),
		(&telegram.Command{Key: deleteCommandKey, Args: []string{sub.Header.String()}}).Button("Delete"),
	}

	html := func(html *html.Writer, chatLink string) *html.Writer {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🛑\n" + sub.Error.String)
	}

	l.Counter("suspend", sub.Labels()).Inc()
	return l.notify(ctx, client, sub.FeedID, telegram.InlineKeyboard(buttons), html)
}

func (l *Event) OnDelete(ctx context.Context, client telegram.Client, sub *feed.Subscription) error {
	html := func(html *html.Writer, chatLink string) *html.Writer {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🗑")
	}

	l.Counter("delete", sub.Labels()).Inc()
	return l.notify(ctx, client, sub.FeedID, nil, html)
}

func (l *Event) OnClear(ctx context.Context, client telegram.Client, feedID telegram.ID, pattern string, deleted int64) error {
	html := func(html *html.Writer, chatLink string) *html.Writer {
		return html.Text(fmt.Sprintf("%d subs @ ", deleted)).
			MarkupString(chatLink).
			Text(" 🗑 (" + pattern + ")")
	}

	l.Counter("clear", (&feed.Header{FeedID: feedID}).Labels()).Inc()
	return l.notify(ctx, client, feedID, nil, html)
}

func (l *Event) notify(ctx context.Context,
	client telegram.Client, chatID telegram.ID,
	markup telegram.ReplyMarkup, writeHTML writeHTMLWithChatLink) error {

	chatLink, err := l.GetChatLink(ctx, client, chatID)
	if err != nil {
		logrus.WithField("chat_id", chatID).
			Warnf("get chat link: %s", err)
		chatLink = chatID.String()
	} else {
		chatLink = html.Anchor("chat", chatLink)
	}

	return l.NotifyAdmins(ctx, client, chatID, markup, func(html *html.Writer) error {
		writeHTML(html, chatLink)
		return nil
	})
}
