package telegram

import "encoding/json"

type ReplyMarkup interface {
	marker(rm ReplyMarkup)
}

type ForceReply struct {
	ForceReply bool `json:"force_reply"`
	Selective  bool `json:"selective,omitempty"`
}

func (r ForceReply) marker(rm ReplyMarkup) {

}

type InlineKeyboardMarkup struct {
	InlineKeyboard []InlineKeyboardButton `json:"inline_keyboard"`
}

func (r InlineKeyboardMarkup) marker(rm ReplyMarkup) {

}

type InlineKeyboardButton struct {
	Text                         string `json:"text"`
	URL                          string `json:"url,omitempty"`
	CallbackData                 string `json:"callback_data,omitempty"`
	SwitchInlineQuery            string `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
	CallbackGame                 *json.RawMessage
	Pay                          bool `json:"pay,omitempty"`
}

type CallbackQuery struct {
	ID              string   `json:"id"`
	From            User     `json:"from"`
	Message         *Message `json:"message,omitempty"`
	InlineMessageID string   `json:"inline_message_id,omitempty"`
	ChatInstance    string   `json:"chat_instance,omitempty"`
	Data            string   `json:"data,omitempty"`
	GameShortName   string   `json:"game_short_name,omitempty"`
}
