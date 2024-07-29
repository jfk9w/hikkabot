package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

// baseClient represents a flu/http.Request factory.
type baseClient struct {
	client   httpf.Client
	endpoint endpointFunc
}

// ValidStatusCodes is a slice of valid API HTTP status codes.
var ValidStatusCodes = []int{
	http.StatusOK,
	http.StatusSeeOther,
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
}

type endpointFunc func(method string) string

// GetUpdates is used to receive incoming updates using long polling.
// An Array of Update objects is returned.
// See https://core.telegram.org/bots/api#getupdates
func (c *baseClient) GetUpdates(ctx context.Context, options GetUpdatesOptions) ([]Update, error) {
	updates := make([]Update, 0)
	return updates, c.Execute(ctx, "getUpdates", flu.JSON(options), &updates)
}

// GetMe is a simple method for testing your bot's auth token. Requires no parameters.
// Returns basic information about the bot in form of a User object.
// See https://core.telegram.org/bots/api#getme
func (c *baseClient) GetMe(ctx context.Context) (*User, error) {
	user := new(User)
	return user, c.Execute(ctx, "getMe", nil, user)
}

func (c *baseClient) ForwardMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *SendOptions) (ID, error) {
	var messageID ID
	form, err := options.body(chatID, ref)
	if err != nil {
		return messageID, err
	}

	return messageID, c.Execute(ctx, "forwardMessage", form, &messageID)
}

func (c *baseClient) CopyMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *CopyOptions) (ID, error) {
	var resp struct {
		MessageID ID `json:"message_id"`
	}

	form, err := options.body(chatID, ref)
	if err != nil {
		return 0, err
	}

	return resp.MessageID, c.Execute(ctx, "copyMessage", form, &resp)
}

// DeleteMessage is used to delete a message, including service messages, with the following limitations:
// - A message can only be deleted if it was sent less than 48 hours ago.
// - Bots can delete outgoing messages in private chats, groups, and supergroups.
// - Bots granted can_post_messages permissions can delete outgoing messages in channels.
// - If the bot is an administrator of a group, it can delete any message there.
// - If the bot has can_delete_messages permission in a supergroup or a updateChannel, it can delete any message there.
// Returns True on success.
// See
//    https://core.telegram.org/bots/api#deletemessage
func (c *baseClient) DeleteMessage(ctx context.Context, ref MessageRef) error {
	var ok bool
	if err := c.Execute(ctx, "deleteMessage", ref.form(), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c *baseClient) EditMessageReplyMarkup(ctx context.Context, ref MessageRef, markup ReplyMarkup) (*Message, error) {
	markupJSON, err := json.Marshal(markup)
	if err != nil {
		return nil, err
	}

	form := ref.form().Set("reply_markup", string(markupJSON))
	message := new(Message)
	if err := c.Execute(ctx, "editMessageReplyMarkup", form, &message); err != nil {
		if tgerr := new(Error); errors.As(err, tgerr) &&
			strings.Contains(tgerr.Description, "message is not modified") {
			return nil, nil
		}

		return nil, err
	}

	return message, nil
}

func (c *baseClient) ExportChatInviteLink(ctx context.Context, chatID ChatID) (string, error) {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam())
	var inviteLink string
	return inviteLink, c.Execute(ctx, "exportChatInviteLink", body, &inviteLink)
}

// GetChat is used to get up to date information about the chat (current name of
// the user for one-on-one conversations, current username of a user, group or updateChannel, etc.).
// Returns a Chat object on success.
// See https://core.telegram.org/bots/api#getchat
func (c *baseClient) GetChat(ctx context.Context, chatID ChatID) (*Chat, error) {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam())
	chat := new(Chat)
	return chat, c.Execute(ctx, "getChat", body, chat)
}

// GetChatAdministrators is used to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information about
// all chat administrators except other bots. If the chat is a group or a supergroup and
// no administrators were appointed, only the creator will be returned.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c *baseClient) GetChatAdministrators(ctx context.Context, chatID ChatID) ([]ChatMember, error) {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam())
	members := make([]ChatMember, 0)
	return members, c.Execute(ctx, "getChatAdministrators", body, &members)
}

func (c *baseClient) GetChatMemberCount(ctx context.Context, chatID ChatID) (int64, error) {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam())

	var count int64
	return count, c.Execute(ctx, "getChatMemberCount", body, &count)
}

// GetChatMember is used to get information about a member of a chat.
// Returns a ChatMember object on success.
// See https://core.telegram.org/bots/api#getchatmember
func (c *baseClient) GetChatMember(ctx context.Context, chatID ChatID, userID ID) (*ChatMember, error) {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam()).
		Set("user_id", userID.queryParam())
	member := new(ChatMember)
	return member, c.Execute(ctx, "getChatMember", body, member)
}

// AnswerCallbackQuery is used to send answers to callback queries sent from inline keyboards.
// The answer will be displayed to the user as a notification at the top of the chat screen or as an alert.
// On success, True is returned.
// https://core.telegram.org/bots/api#answercallbackquery
func (c *baseClient) AnswerCallbackQuery(ctx context.Context, id string, options *AnswerOptions) error {
	var ok bool
	if err := c.Execute(ctx, "answerCallbackQuery", options.body(id), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c *baseClient) SetMyCommands(ctx context.Context, scope *BotCommandScope, commands []BotCommand) error {
	type request struct {
		Commands []BotCommand     `json:"commands"`
		Scope    *BotCommandScope `json:"scope,omitempty"`
	}

	req := request{commands, scope}
	var ok bool
	if err := c.Execute(ctx, "setMyCommands", flu.JSON(req), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c *baseClient) GetMyCommands(ctx context.Context, scope *BotCommandScope) ([]BotCommand, error) {
	type request struct {
		Scope *BotCommandScope `json:"scope,omitempty"`
	}

	req := request{scope}
	resp := make([]BotCommand, 0)
	return resp, c.Execute(ctx, "getMyCommands", flu.JSON(req), &resp)
}

func (c *baseClient) DeleteMyCommands(ctx context.Context, scope *BotCommandScope) error {
	type request struct {
		Scope *BotCommandScope `json:"scope,omitempty"`
	}

	req := request{scope}
	var ok bool
	if err := c.Execute(ctx, "deleteMyCommands", flu.JSON(req), &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c *baseClient) SendChatAction(ctx context.Context, chatID ChatID, action string) error {
	body := new(httpf.Form).
		Set("chat_id", chatID.queryParam()).
		Set("action", action)
	var ok bool
	if err := c.Execute(ctx, "sendChatAction", body, &ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("not ok")
	}

	return nil
}

func (c *baseClient) Execute(ctx context.Context, method string, body flu.EncoderTo, resp interface{}) error {
	err := httpf.POST(c.endpoint(method), body).
		Exchange(ctx, c.client).
		DecodeBody(newResponse(resp)).
		CheckStatus(ValidStatusCodes...).
		Error()
	log().Resultf(ctx, logf.Trace, logf.Warn, "execute [%s]: %v", method, err)
	return err
}
