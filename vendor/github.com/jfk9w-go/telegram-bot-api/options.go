package telegram

import (
	"encoding/json"

	httpf "github.com/jfk9w-go/flu/httpf"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// GetUpdatesOptions is /getUpdates request options.
// See https://core.telegram.org/bots/api#getupdates
type GetUpdatesOptions struct {
	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers of previously received updates.
	// By default, updates starting with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called with an offset
	// higher than its update_id. The negative offset can be specified to retrieve updates
	// starting from -offset update from the end of the updates queue.
	// All previous updates will be forgotten.
	Offset ID `json:"offset,omitempty"`
	// Limits the number of updates to be retrieved.
	// Values between 1—100 are accepted. Defaults to 100.
	Limit int `json:"limit,omitempty"`
	// Timeout for long polling.
	TimeoutSecs int `json:"timeout,omitempty"`
	// List the types of updates you want your bot to receive.
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

type SendOptions struct {
	DisableNotification bool
	ReplyToMessageID    ID
	ReplyMarkup         ReplyMarkup
}

func (o *SendOptions) body(chatID ChatID, item sendable) (flu.EncoderTo, error) {
	mediaGroup := item.kind() == "mediaGroup"
	var form *httpf.Form
	if !mediaGroup {
		form = httpf.FormValue(item)
	} else {
		form = new(httpf.Form)
	}

	form = form.Set("chat_id", chatID.queryParam())
	if o != nil {
		if o.DisableNotification {
			form = form.Set("disable_notification", "1")
		}
		if o.ReplyToMessageID != 0 {
			form = form.Set("reply_to_message_id", o.ReplyToMessageID.queryParam())
		}
		if !mediaGroup && o.ReplyMarkup != nil {
			bytes, err := json.Marshal(o.ReplyMarkup)
			if err != nil {
				return nil, errors.Wrap(err, "serialize reply_markup")
			}
			form = form.Set("reply_markup", string(bytes))
		}
	}
	return item.body(form)
}

type CopyOptions struct {
	*SendOptions
	Caption   string    `url:"caption,omitempty"`
	ParseMode ParseMode `url:"parse_mode,omitempty"`
}

func (o *CopyOptions) body(chatID ChatID, ref MessageRef) (flu.EncoderTo, error) {
	form := httpf.FormValue(o)
	form.Set("chat_id", chatID.queryParam())
	return ref.body(form)
}

type AnswerOptions struct {
	Text      string `url:"text,omitempty"`
	ShowAlert bool   `url:"show_alert,omitempty"`
	URL       string `url:"url,omitempty"`
	CacheTime int    `url:"cache_time,omitempty"`
}

func (o *AnswerOptions) body(id string) flu.EncoderTo {
	return httpf.FormValue(o).Set("callback_query_id", id)
}
