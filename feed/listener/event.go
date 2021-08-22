package listener

import (
	"context"
	"fmt"

	"github.com/jfk9w-go/flu/metrics"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/sirupsen/logrus"
)

type writeHTMLWithChatLink func(html *richtext.HTMLWriter, chatLink string) *richtext.HTMLWriter

type Event struct {
	AccessControl
	metrics.Registry
	telegram.Client
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
		chatLink = richtext.HTMLAnchor("chat", chatLink)
	}

	return l.NotifyAdmins(ctx, l, chatID, markup, func(html *richtext.HTMLWriter) error {
		writeHTML(html, chatLink)
		return nil
	})
}

func (l *Event) OnResume(ctx context.Context, sub *feed.Subscription) error {
	buttons := []telegram.Button{
		telegram.Command{Key: suspendCommandKey, Args: []string{sub.Header.String()}}.Button("Suspend"),
	}

	html := func(html *richtext.HTMLWriter, chatLink string) *richtext.HTMLWriter {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🔥")
	}

	l.Counter("resume", metrics.Labels(sub.Fields())).Inc()
	return l.notify(ctx, l.Client, sub.FeedID, telegram.InlineKeyboard(buttons), html)
}

func (l *Event) OnSuspend(ctx context.Context, sub *feed.Subscription) error {
	buttons := []telegram.Button{
		telegram.Command{Key: resumeCommandKey, Args: []string{sub.Header.String()}}.Button("Resume"),
		telegram.Command{Key: deleteCommandKey, Args: []string{sub.Header.String()}}.Button("Delete"),
	}

	html := func(html *richtext.HTMLWriter, chatLink string) *richtext.HTMLWriter {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🛑\n" + sub.Error.String)
	}

	l.Counter("suspend", metrics.Labels(sub.Fields())).Inc()
	return l.notify(ctx, l.Client, sub.FeedID, telegram.InlineKeyboard(buttons), html)
}

func (l *Event) OnDelete(ctx context.Context, sub *feed.Subscription) error {
	html := func(html *richtext.HTMLWriter, chatLink string) *richtext.HTMLWriter {
		return html.Text(sub.Name + " @ ").
			MarkupString(chatLink).
			Text(" 🗑")
	}

	l.Counter("delete", metrics.Labels(sub.Fields())).Inc()
	return l.notify(ctx, l.Client, sub.FeedID, nil, html)
}

func (l *Event) OnClear(ctx context.Context, feedID telegram.ID, pattern string, deleted int64) error {
	html := func(html *richtext.HTMLWriter, chatLink string) *richtext.HTMLWriter {
		return html.Text(fmt.Sprintf("%d subs @ ", deleted)).
			MarkupString(chatLink).
			Text(" 🗑 (" + pattern + ")")
	}

	l.Counter("clear", metrics.Labels{"feedID": feedID}).Add(float64(deleted))
	return l.notify(ctx, l.Client, feedID, nil, html)
}
