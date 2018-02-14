package telegram

import (
	"encoding/json"
	"net/url"
	"strconv"
)

const (
	Markdown = "Markdown"
	HTML     = "HTML"
)

type (
	Request interface {
		Method() string
		Parameters() url.Values
	}

	ReplyMarkup interface {
		marker(rm ReplyMarkup)
	}
)

type GenericRequest struct {
	method string
	params map[string]string
}

func (r GenericRequest) Method() string {
	return r.method
}

func (r GenericRequest) Parameters() url.Values {
	p := url.Values{}
	for k, v := range r.params {
		p.Set(k, v)
	}

	return p
}

type GetUpdatesRequest struct {
	Offset         int
	Limit          int
	Timeout        int
	AllowedUpdates []string
}

func (r GetUpdatesRequest) Method() string {
	return "getUpdates"
}

func (r GetUpdatesRequest) Parameters() url.Values {
	v := url.Values{}
	if r.Offset > 0 {
		v.Set("offset", strconv.Itoa(r.Offset))
	}
	if r.Limit > 0 {
		v.Set("limit", strconv.Itoa(r.Limit))
	}
	if r.Timeout > 0 {
		v.Set("timeout", strconv.Itoa(r.Timeout))
	}
	for _, au := range r.AllowedUpdates {
		v.Add("allowed_updates", au)
	}
	return v
}

type ChatRef struct {
	ID       ChatID `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
}

func (r ChatRef) Parameters() url.Values {
	return url.Values{
		"chat_id": []string{r.Key()},
	}
}

func (r ChatRef) Key() string {
	if len(r.Username) > 0 {
		return r.Username
	} else {
		return FormatChatID(r.ID)
	}
}

func (r ChatRef) IsChannel() bool {
	return len(r.Username) > 0
}

func FormatChatID(chatId ChatID) string {
	return strconv.FormatInt(int64(chatId), 10)
}

func ParseChatID(value string) ChatID {
	chatId, _ := strconv.ParseInt(value, 10, 64)
	return ChatID(chatId)
}

type SendMessageRequest struct {
	Chat                  ChatRef
	Text                  string
	ParseMode             string
	DisableWebPagePreview bool
	DisableNotification   bool
	ReplyToMessageID      MessageID
	ReplyMarkup           ReplyMarkup
}

func (r SendMessageRequest) Method() string {
	return "sendMessage"
}

func (r SendMessageRequest) Parameters() url.Values {
	v := r.Chat.Parameters()
	v.Set("text", r.Text)
	if len(r.ParseMode) > 0 {
		v.Set("parse_mode", r.ParseMode)
	}
	if r.DisableWebPagePreview {
		v.Set("disable_web_page_preview", "true")
	}
	if r.DisableNotification {
		v.Set("disable_notification", "true")
	}
	if r.ReplyToMessageID != 0 {
		v.Set("reply_to_message_id", strconv.Itoa(int(r.ReplyToMessageID)))
	}
	if r.ReplyMarkup != nil {
		rm, err := json.Marshal(r.ReplyMarkup)
		if err == nil {
			v.Set("reply_markup", string(rm))
		}
	}
	return v
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

type (
	InlineKeyboardButton struct {
		Text                         string `json:"text"`
		URL                          string `json:"url,omitempty"`
		CallbackData                 string `json:"callback_data,omitempty"`
		SwitchInlineQuery            string `json:"switch_inline_query,omitempty"`
		SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
		CallbackGame                 *json.RawMessage
		Pay                          bool `json:"pay,omitempty"`
	}

	CallbackQuery struct {
		ID              string   `json:"id"`
		From            User     `json:"from"`
		Message         *Message `json:"message,omitempty"`
		InlineMessageID string   `json:"inline_message_id,omitempty"`
		ChatInstance    string   `json:"chat_instance,omitempty"`
		Data            string   `json:"data,omitempty"`
		GameShortName   string   `json:"game_short_name,omitempty"`
	}
)
