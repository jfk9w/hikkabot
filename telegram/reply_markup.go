package telegram

import "encoding/json"

type ReplyMarkup interface {
	marker(rm ReplyMarkup)
}

// Upon receiving a message with this object,
// Telegram clients will display a reply interface to the user
// (act as if the user has selected the bot‘s message and tapped ’Reply').
// This can be extremely useful if you want to create user-friendly
// step-by-step interfaces without having to sacrifice privacy mode.
type ForceReply struct {

	// Shows reply interface to the user, as if they manually selected the bot‘s message and tapped ’Reply'
	// Must be true
	ForceReply bool `json:"force_reply"`

	// Optional. Use this parameter if you want to force reply from specific users only.
	// Targets:
	// 1) users that are @mentioned in the text of the Message object;
	// 2) if the bot's message is a reply (has reply_to_message_id), sender of the original message.
	Selective bool `json:"selective,omitempty"`
}

func (r ForceReply) marker(rm ReplyMarkup) {

}

// This object represents an inline keyboard that appears right next to the message it belongs to.
type InlineKeyboardMarkup struct {

	// Array of button rows, each represented by an Array of InlineKeyboardButton objects
	InlineKeyboard []InlineKeyboardButton `json:"inline_keyboard"`
}

func (r InlineKeyboardMarkup) marker(rm ReplyMarkup) {

}

// This object represents one button of an inline keyboard. You must use exactly one of the optional fields.
type InlineKeyboardButton struct {

	// Label text on the button
	Text string `json:"text"`

	// Optional. HTTP url to be opened when button is pressed
	URL string `json:"url,omitempty"`

	// Optional. Data to be sent in a callback query to the bot when button is pressed, 1-64 bytes
	CallbackData string `json:"callback_data,omitempty"`

	// Optional. If set, pressing the button will prompt the user
	// to select one of their chats, open that chat and insert the bot‘s username
	// and the specified inline query in the input field. Can be empty,
	// in which case just the bot’s username will be inserted.
	//
	// Note: This offers an easy way for users to start using your bot in inline mode
	// when they are currently in a private chat with it.
	// Especially useful when combined with switch_pm… actions –
	// in this case the user will be automatically returned to the chat they switched from,
	// skipping the chat selection screen.
	SwitchInlineQuery string `json:"switch_inline_query,omitempty"`

	// Optional. If set, pressing the button will insert the bot‘s username
	// and the specified inline query in the current chat's input field.
	// Can be empty, in which case only the bot’s username will be inserted.
	//
	// This offers a quick way for the user to open your bot in inline mode
	// in the same chat – good for selecting something from multiple options.
	SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`

	// Optional. Description of the game that will be launched when the user presses the button.
	//
	// NOTE: This type of button must always be the first button in the first row.
	CallbackGame *json.RawMessage

	// Optional. Specify True, to send a Pay button.
	//
	// NOTE: This type of button must always be the first button in the first row.
	Pay bool `json:"pay,omitempty"`
}

// This object represents an incoming callback query from a callback button
// in an inline keyboard. If the button that originated the query was attached
// to a message sent by the bot, the field message will be present.
// If the button was attached to a message sent via the bot (in inline mode),
// the field inline_message_id will be present.
// Exactly one of the fields data or game_short_name will be present.
type CallbackQuery struct {

	// Unique identifier for this query
	ID string `json:"id"`

	// Sender
	From User `json:"from"`

	// Optional. Message with the callback button that originated the query.
	// Note that message content and message date will not be available if the message is too old
	Message *Message `json:"message,omitempty"`

	// Optional. Identifier of the message sent via the bot in inline mode, that originated the query.
	InlineMessageID string `json:"inline_message_id,omitempty"`

	// Global identifier, uniquely corresponding to the chat
	// to which the message with the callback button was sent. Useful for high scores in games.
	ChatInstance string `json:"chat_instance,omitempty"`

	// Optional. Data associated with the callback button.
	// Be aware that a bad client can send arbitrary data in this field.
	Data string `json:"data,omitempty"`

	// Optional. Short name of a Game to be returned, serves as the unique identifier for the game
	GameShortName string `json:"game_short_name,omitempty"`
}
