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

// Use this method to receive incoming updates using long polling (wiki).
// An Array of Update objects is returned.
type GetUpdatesRequest struct {

	// Identifier of the first update to be returned.
	// Must be greater by one than the highest among the identifiers
	// of previously received updates. By default, updates starting
	// with the earliest unconfirmed update are returned.
	// An update is considered confirmed as soon as getUpdates is called
	// with an offset higher than its update_id.
	// The negative offset can be specified to retrieve updates
	// starting from -offset update from the end of the updates queue.
	// All previous updates will forgotten.
	Offset int

	// Limits the number of updates to be retrieved. Values between 1—100 are accepted. Defaults to 100.
	Limit int

	// Timeout in seconds for long polling. Defaults to 0, i.e. usual short polling.
	// Should be positive, short polling should be used for testing purposes only.
	Timeout int

	// List the types of updates you want your bot to receive.
	// For example, specify [“message”, “edited_channel_post”, “callback_query”]
	// to only receive updates of these types.
	// See Update for a complete list of available update types.
	// Specify an empty list to receive all updates regardless of type (default).
	// If not specified, the previous setting will be used.
	//
	// Please note that this parameter doesn't affect updates created
	// before the call to the getUpdates,
	// so unwanted updates may be received for a short period of time.
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

// A simple method for testing your bot's auth token.
// Requires no parameters. Returns basic information about the bot in form of a User object.
type GetMeRequest struct{}

func (r GetMeRequest) Method() string {
	return "getMe"
}

func (r GetMeRequest) Parameters() url.Values {
	return url.Values{}
}

// Unique identifier for the target chat or
// username of the target channel (in the format @channelusername)
type ChatRef struct {
	ID       ChatID
	Username string
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
		return strconv.FormatInt(int64(r.ID), 10)
	}
}

const (
	Markdown = "Markdown"
	HTML     = "HTML"
)

// Use this method to send text messages. On success, the sent Message is returned.
type SendMessageRequest struct {
	Chat ChatRef

	// Text of the message to be sent
	Text string

	// Send Markdown or HTML, if you want Telegram apps
	// to show bold, italic, fixed-width text or inline URLs in your bot's message.
	ParseMode string

	// Disables link previews for links in this message
	DisableWebPagePreview bool

	// Sends the message silently. Users will receive a notification with no sound.
	DisableNotification bool

	// If the message is a reply, ID of the original message
	ReplyToMessageID MessageID

	// Additional interface options.
	// A JSON-serialized object for an inline keyboard, custom reply keyboard,
	// instructions to remove reply keyboard or to force a reply from the user.
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

// Use this method to change the title of a chat. Titles can't be changed for private chats.
// The bot must be an administrator in the chat for this to work
// and must have the appropriate admin rights. Returns True on success.
//
// Note: In regular groups (non-supergroups),
// this method will only work if the ‘All Members Are Admins’ setting is off in the target group.
type SetChatTitleRequest struct {
	Chat ChatRef

	// New chat title, 1-255 characters
	Title string
}

func (r SetChatTitleRequest) Method() string {
	return "setChatTitle"
}

func (r SetChatTitleRequest) Parameters() url.Values {
	v := r.Chat.Parameters()
	v.Set("title", r.Title)
	return v
}
