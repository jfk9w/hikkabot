package telegram

import "github.com/jfk9w-go/hikkabot/common/telegram-bot-api/api"

type (
	// API
	API       = api.T
	APIConfig = api.Config

	// Types
	MessageID     = api.MessageID
	Ref           = api.Ref
	ChatID        = api.ChatID
	Username      = api.Username
	ChatType      = api.ChatType
	User          = api.User
	Chat          = api.Chat
	Message       = api.Message
	MessageEntity = api.MessageEntity
	Update        = api.Update
	ChatMember    = api.ChatMember

	// Errors
	Error           = api.Error
	TooManyMessages = api.TooManyMessages

	// Opts
	UpdatesOpts = api.UpdatesOpts
	SendOpts    = api.SendOpts
	MessageOpts = api.MessageOpts
	MediaOpts   = api.MediaOpts
	VideoOpts   = api.VideoOpts
)

const (
	None     = api.None
	HTML     = api.HTML
	Markdown = api.Markdown

	PrivateChatType = api.PrivateChat
	GroupType       = api.Group
	SupergroupType  = api.Supergroup
	ChannelType     = api.Channel

	MaxPhotoSize   = 10 * 1024 * 1024
	MaxVideoSize   = 50 * 1024 * 1024
	MaxMessageSize = api.MaxMessageSize
)

func NewAPI(config APIConfig) *API {
	return api.New(config)
}

func ParseChatID(value string) (ChatID, error) {
	return api.ParseChatID(value)
}

func ParseUsername(value string) (Username, error) {
	return api.ParseUsername(value)
}

func ParseRef(value string) (Ref, error) {
	var username, err = ParseUsername(value)
	if err == nil {
		return username, nil
	}

	return ParseChatID(value)
}

type Config struct {
	APIConfig
	RouterConfig
}

type T struct {
	*Router
	*Updater
}

func Configure(config Config, updates *UpdatesOpts) *T {
	var (
		api     = NewAPI(config.APIConfig)
		sink    = Route(api, config.RouterConfig)
		updater = RunUpdater(api, updates)
	)

	return &T{sink, updater}
}
