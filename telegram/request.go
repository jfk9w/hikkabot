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

type GetUpdatesRequest struct {
	offset         int
	limit          int
	timeout        int
	allowedUpdates []string
}

func (r GetUpdatesRequest) Method() string {
	return "/getUpdates"
}

func (r GetUpdatesRequest) Parameters() url.Values {
	v := url.Values{}
	if r.offset > 0 {
		v.Set("offset", strconv.Itoa(r.offset))
	}
	if r.limit > 0 {
		v.Set("limit", strconv.Itoa(r.limit))
	}
	if r.timeout > 0 {
		v.Set("timeout", strconv.Itoa(r.timeout))
	}
	for _, au := range r.allowedUpdates {
		v.Add("allowed_updates", au)
	}
	return v
}

type GetMeRequest struct{}

func (r GetMeRequest) Method() string {
	return "/getMe"
}

func (r GetMeRequest) Parameters() url.Values {
	return url.Values{}
}

type ChatRequestField struct {
	ID       ChatID
	Username string
}

func (r ChatRequestField) Parameters() url.Values {
	v := url.Values{}
	if len(r.Username) > 0 {
		v.Set("chat_id", r.Username)
	} else {
		v.Set("chat_id", strconv.FormatInt(int64(r.ID), 10))
	}
	return v
}

const (
	ParseModeMarkdown = "Markdown"
	ParseModeHTML     = "HTML"
)

type SendMessageRequest struct {
	Chat                  ChatRequestField
	Text                  string
	ParseMode             string
	DisableWebPagePreview bool
	DisableNotification   bool
	ReplyToMessageID      MessageID
	ReplyMarkup
}

func (r SendMessageRequest) Method() string {
	return "/sendMessage"
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
		v.Set("disable_web_page_preview", "true")
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
