package telegram

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type Request interface {
	Method() string
	Parameters() url.Values
}

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
	Username string `json:"username,omitempty`
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

const (
	Markdown = "Markdown"
	HTML     = "HTML"
)

type SendMessageRequest struct {
	Chat                  ChatRef
	Text                  string
	ParseMode             string
	DisableWebPagePreview bool
	DisableNotification   bool
	ReplyToMessageID      MessageID
	ReplyMarkup
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
