package telegram

type MessageID int
type ChatID int64
type UserID int

// This object represents a Telegram user or bot.
type User struct {
	// Unique identifier for this user or bot
	ID UserID `json:"id"`

	// True, if this user is a bot
	IsBot bool `json:"is_bot"`

	// User‘s or bot’s first name
	FirstName string `json:"first_name"`

	// Optional. User‘s or bot’s last name
	LastName string `json:"last_name"`

	// Optional. User‘s or bot’s username
	Username string `json:"username"`

	// Optional. IETF language tag of the user's language
	LanguageCode string `json:"language_code"`
}

// This object represents a chat.
type Chat struct {
	// Unique identifier for this chat.
	// This number may be greater than 32 bits and some programming languages
	// may have difficulty/silent defects in interpreting it. But it is smaller
	// than 52 bits, so a signed 64 bit integer or double-precision float type
	// are safe for storing this identifier.
	ID ChatID `json:"id"`

	// Type of chat, can be either “private”, “group”, “supergroup” or “channel”
	Type string `json:"type"`

	// Optional. Title, for supergroups, channels and group chats
	Title string `json:"title"`

	// Optional. Username, for private chats, supergroups and channels if available
	Username string `json:"username"`

	// Optional. First name of the other party in a private chat
	FirstName string `json:"first_name"`

	// Optional. Last name of the other party in a private chat
	LastName string `json:"last_name"`

	// Optional. True if a group has ‘All Members Are Admins’ enabled.
	AllMembersAreAdministrators bool `json:"all_members_are_administrators"`

	// Optional. Chat photo. Returned only in getChat.
	// Photo ChatPhoto `json:"photo"`

	// Optional. Description, for supergroups and channel chats. Returned only in getChat.
	Description string `json:"description"`

	// Optional. Chat invite link, for supergroups and channel chats. Returned only in getChat.
	InviteLink string `json:"invite_link"`

	// Optional. Pinned message, for supergroups. Returned only in getChat.
	// PinnedMessage Message `json:"pinned_message"`

	// Optional. For supergroups, name of group sticker set. Returned only in getChat.
	StickerSetName string `json:"sticker_set_name"`

	// Optional. True, if the bot can change the group sticker set. Returned only in getChat.
	CanSetStickerSet bool `json:"can_set_sticker_set"`
}

// This object represents a message.
type Message struct {
	// Unique message identifier inside this chat
	ID MessageID `json:"message_id"`

	// Optional. Sender, empty for messages sent to channels
	From User `json:"from"`

	// Date the message was sent in Unix time
	Date int `json:"date"`

	// Conversation the message belongs to
	Chat `json:"chat"`

	// Optional. For forwarded messages, sender of the original message
	ForwardFrom User `json:"forward_from"`

	// Optional. For messages forwarded from channels, information about the original channel
	ForwardFromChat Chat `json:"forward_from_chat"`

	// Optional. For messages forwarded from channels, identifier of the original message in the channel
	ForwardFromMessageID int `json:"forward_from_message_id"`

	// Optional. For messages forwarded from channels, signature of the post author if present
	ForwardSignature string `json:"forward_signature"`

	// Optional. For forwarded messages, date the original message was sent in Unix time
	ForwardDate int `json:"forward_date"`

	// Optional. For replies, the original message. Note that the Message object
	// in this field will not contain further reply_to_message fields even if it itself is a reply.
	ReplyToMessage *Message `json:"reply_to_message"`

	// Optional. Date the message was last edited in Unix time
	EditDate int `json:"edit_date"`

	// Optional. Signature of the post author for messages in channels
	AuthorSignature string `json:"author_signature"`

	// Optional. For text messages, the actual UTF-8 text of the message, 0-4096 characters.
	Text string `json:"text"`

	// Optional. For text messages, special entities like usernames,
	// URLs, bot commands, etc. that appear in the text
	Entities []MessageEntity `json:"entities"`

	// Optional. For messages with a caption, special entities like usernames,
	// URLs, bot commands, etc. that appear in the caption
	CaptionEntities []MessageEntity `json:"caption_entities"`
}

// This object represents one special entity in a text message.
// For example, hashtags, usernames, URLs, etc.
type MessageEntity struct {
	// Type of the entity. Can be mention (@username), hashtag, bot_command,
	// url, email, bold (bold text), italic (italic text),
	// code (monowidth string), pre (monowidth block),
	// text_link (for clickable text URLs), text_mention (for users without usernames)
	Type string `json:"type"`

	// Offset in UTF-16 code units to the start of the entity
	Offset int `json:"offset"`

	// Length of the entity in UTF-16 code units
	Length int `json:"length"`

	// Optional. For “text_link” only, url that will be opened after user taps on the text
	URL string `json:"url"`

	// Optional. For “text_mention” only, the mentioned user
	User `json:"user"`
}

// This object represents an incoming update.
// At most one of the optional parameters can be present in any given update.
type Update struct {
	// The update‘s unique identifier. Update identifiers start from a certain
	// positive number and increase sequentially. This ID becomes especially
	// handy if you’re using Webhooks, since it allows you to ignore repeated
	// updates or to restore the correct update sequence, should they get out of order.
	ID int `json:"update_id"`

	// Optional. New incoming message of any kind — text, photo, sticker, etc.
	Message `json:"message"`

	// Optional. New version of an incoming message of any kind — text, photo, sticker, etc.
	EditedMessage Message `json:"edited_message"`

	// Optional. New incoming channel post of any kind — text, photo, sticker, etc.
	ChannelPost Message `json:"channel_post"`

	// Optional. New version of a channel post that is known to the bot and was edited
	EditedChannelPost Message `json:"edited_message_post"`
}
