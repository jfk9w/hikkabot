package access

import (
	"context"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/pkg/errors"
)

type DefaultControl struct {
	userIDs     map[telegram.ID]bool
	inviteLinks map[telegram.ID]string
	mu          flu.RWMutex
}

func NewDefaultControl(userIDs ...telegram.ID) *DefaultControl {
	userIDMap := make(map[telegram.ID]bool)
	for _, userID := range userIDs {
		userIDMap[userID] = true
	}

	return &DefaultControl{
		userIDs:     userIDMap,
		inviteLinks: make(map[telegram.ID]string),
	}
}

func (c *DefaultControl) getChatLinkFromCache(chatID telegram.ID) (string, bool) {
	defer c.mu.RLock().Unlock()
	link, ok := c.inviteLinks[chatID]
	return link, ok
}

func (c *DefaultControl) GetChatLink(ctx context.Context, client telegram.Client, chatID telegram.ID) (string, error) {
	if chatID > 0 {
		return "tg://resolve?domain=" + client.Username(), nil
	}

	if inviteLink, ok := c.getChatLinkFromCache(chatID); ok {
		return inviteLink, nil
	}

	defer c.mu.Lock().Unlock()
	inviteLink, ok := c.inviteLinks[chatID]
	if ok {
		return inviteLink, nil
	}

	chat, err := client.GetChat(ctx, chatID)
	if err != nil {
		return "", errors.Wrap(err, "get chat")
	}

	inviteLink = chat.InviteLink
	if inviteLink == "" {
		if chat.Username != nil {
			inviteLink = "https://t.me/" + chat.Username.String()
		} else {
			inviteLink, err = client.ExportChatInviteLink(ctx, chatID)
			if err != nil {
				return "", errors.Wrap(err, "export chat invite link")
			}
		}
	}

	c.inviteLinks[chatID] = inviteLink
	return inviteLink, nil
}

func (c *DefaultControl) CheckAccess(ctx context.Context,
	client telegram.Client, userID, chatID telegram.ID) (
	context.Context, error) {

	if _, ok := c.userIDs[userID]; !ok {
		return nil, feed.ErrForbidden
	} else {
		return ctx, nil
	}
}

func (c *DefaultControl) NotifyAdmins(ctx context.Context,
	client telegram.Client, chatID telegram.ID,
	markup telegram.ReplyMarkup, writeHTML feed.WriteHTML) error {

	buffer := richtext.NewBufferedOutput()
	html := richtext.HTMLWithTransport(ctx, buffer)
	if err := writeHTML(html); err != nil {
		return errors.Wrap(err, "write HTML")
	}

	if err := html.Flush(); err != nil {
		return errors.Wrap(err, "flush HTML")
	}

	lastIdx := len(buffer.Pages) - 1
	pages := buffer.Pages
	userIDs := make([]telegram.ChatID, len(c.userIDs))
	i := 0
	for userID, _ := range c.userIDs {
		userIDs[i] = userID
		i++
	}

	output := &richtext.TelegramOutput{
		Sender:  client,
		ChatIDs: userIDs,
		Strict:  true,
	}

	for i, page := range pages {
		markup := markup
		if i != lastIdx {
			markup = nil
		}

		ctx = richtext.WithParseMode(richtext.WithReplyMarkup(ctx, markup), telegram.HTML)
		if err := output.Text(ctx, page, true); err != nil {
			return err
		}
	}

	return nil
}
