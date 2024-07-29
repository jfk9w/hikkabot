package telegram

import (
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
)

// ParseMode is a parse_mode request parameter type.
type ParseMode string

const (
	// None is used for empty parse_mode.
	None ParseMode = ""
	// Markdown is "Markdown" parse_mode value.
	Markdown ParseMode = "Markdown"
	// HTML is "HTML" parse_mode value.
	HTML ParseMode = "HTML"

	// MaxMessageSize is maximum message character length.
	MaxMessageSize = 4096
	// MaxCaptionSize is maximum caption character length.
	MaxCaptionSize = 1024
)

// ChatType can be either “private”, “group”, “supergroup” or “channel”
type ChatType string

const (
	PrivateChat ChatType = "private"
	GroupChat   ChatType = "group"
	Supergroup  ChatType = "supergroup"
	Channel     ChatType = "channel"
)

func (t ChatType) SendDelay() time.Duration {
	return SendDelays[t]
}

type BotCommandScopeType string

const (
	BotCommandScopeDefault               BotCommandScopeType = "default"
	BotCommandScopeAllPrivateChats       BotCommandScopeType = "all_private_chats"
	BotCommandScopeAllGroupChats         BotCommandScopeType = "all_group_chats"
	BotCommandScopeAllChatAdministrators BotCommandScopeType = "all_chat_administrators"
	BotCommandScopeChat                  BotCommandScopeType = "chat"
	BotCommandScopeChatAdministrators    BotCommandScopeType = "chat_administrators"
	BotCommandScopeChatMember            BotCommandScopeType = "chat_member"
)

type (
	// User (https://core.telegram.org/bots/api#user)
	User struct {
		ID        ID        `json:"id"`
		IsBot     bool      `json:"is_bot"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Username  *Username `json:"username"`
	}

	// Chat (https://core.telegram.org/bots/api#chat)
	Chat struct {
		ID                          ID        `json:"id"`
		Type                        ChatType  `json:"type"`
		Title                       string    `json:"title"`
		Username                    *Username `json:"username"`
		FirstName                   string    `json:"first_name"`
		LastName                    string    `json:"last_name"`
		AllMembersAreAdministrators bool      `json:"all_members_are_administrators"`
		InviteLink                  string    `json:"invite_link"`
	}

	MessageFile struct {
		ID string `json:"file_id"`
	}

	// Message (https://core.telegram.org/bots/api#message)
	Message struct {
		ID             ID              `json:"message_id"`
		From           User            `json:"from"`
		Date           int             `json:"date"`
		Chat           Chat            `json:"chat"`
		Text           string          `json:"text"`
		Entities       []MessageEntity `json:"entities"`
		ReplyToMessage *Message        `json:"reply_to_message"`
		Photo          []MessageFile   `json:"photo"`
		Video          *MessageFile    `json:"video"`
		Animation      *MessageFile    `json:"animation"`
	}

	// MessageRef is used for message copying and forwarding.
	MessageRef struct {
		ChatID ChatID `url:"from_chat_id"`
		ID     ID     `url:"message_id"`
	}

	// MessageEntity (https://core.telegram.org/bots/api#messageentity)
	MessageEntity struct {
		Type   string `json:"type"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
		URL    string `json:"url"`
		User   *User  `json:"user"`
	}

	// Update (https://core.telegram.org/bots/api#update)
	Update struct {
		ID                ID             `json:"update_id"`
		Message           *Message       `json:"message"`
		EditedMessage     *Message       `json:"edited_message"`
		ChannelPost       *Message       `json:"channel_post"`
		EditedChannelPost *Message       `json:"edited_message_post"`
		CallbackQuery     *CallbackQuery `json:"callback_query"`
	}

	// ChatMember (https://core.telegram.org/bots/api#chatmember)
	ChatMember struct {
		User   User   `json:"user"`
		Status string `json:"status"`
	}

	// CallbackQuery (https://core.telegram.org/bots/api#callbackquery)
	CallbackQuery struct {
		ID              string   `json:"id"`
		From            User     `json:"from"`
		Message         *Message `json:"message"`
		InlineMessageID *string  `json:"inline_message_id"`
		ChatInstance    *string  `json:"chat_instance"`
		Data            *string  `json:"data"`
		GameShortName   *string  `json:"game_short_name"`
	}

	// InlineKeyboardButton (https://core.telegram.org/bots/api#inlinekeyboardbutton)
	InlineKeyboardButton struct {
		Text                         string `json:"text"`
		URL                          string `json:"url,omitempty"`
		CallbackData                 string `json:"callback_data,omitempty"`
		SwitchInlineQuery            string `json:"switch_inline_query,omitempty"`
		SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
	}

	ReplyMarkup interface {
		self() ReplyMarkup
	}

	ForceReply struct {
		ForceReply bool `json:"force_reply,omitempty"`
		Selective  bool `json:"selective,omitempty"`
	}

	// InlineKeyboardMarkup (https://core.telegram.org/bots/api#inlinekeyboardmarkup)
	InlineKeyboardMarkup struct {
		InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
	}

	BotCommandScope struct {
		Type   BotCommandScopeType `json:"type"`
		ChatID ChatID              `json:"chat_id,omitempty"`
		UserID ID                  `json:"user_id,omitempty"`
	}

	BotCommand struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
)

func (m *Message) Ref() MessageRef {
	return MessageRef{
		ChatID: m.Chat.ID,
		ID:     m.ID,
	}
}

func (r MessageRef) kind() string {
	return "__internal__"
}

func (r MessageRef) body(form *httpf.Form) (flu.EncoderTo, error) {
	return form.
		Set("from_chat_id", r.ChatID.queryParam()).
		Set("message_id", r.ID.queryParam()), nil
}

func (r MessageRef) form() *httpf.Form {
	return new(httpf.Form).
		Set("chat_id", r.ChatID.queryParam()).
		Set("message_id", r.ID.queryParam())
}

func (r ForceReply) self() ReplyMarkup {
	return r
}

func (m InlineKeyboardMarkup) self() ReplyMarkup {
	return m
}
