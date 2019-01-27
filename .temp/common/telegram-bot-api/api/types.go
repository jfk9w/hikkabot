package api

type ChatType string

const (
	PrivateChat ChatType = "private"
	Group       ChatType = "group"
	Supergroup  ChatType = "supergroup"
	Channel     ChatType = "channel"
)

type (
	User struct {
		ID        ChatID   `json:"id"`
		IsBot     bool     `json:"is_bot"`
		FirstName string   `json:"first_name"`
		LastName  string   `json:"last_name"`
		Username  Username `json:"username"`
	}

	Chat struct {
		ID                          ChatID   `json:"id"`
		Type                        ChatType `json:"type"`
		Title                       string   `json:"title"`
		Username                    Username `json:"username"`
		FirstName                   string   `json:"first_name"`
		LastName                    string   `json:"last_name"`
		AllMembersAreAdministrators bool     `json:"all_members_are_administrators"`
	}

	Message struct {
		ID       MessageID       `json:"message_id"`
		From     User            `json:"from"`
		Date     int             `json:"date"`
		Chat     Chat            `json:"chat"`
		Text     string          `json:"text"`
		Entities []MessageEntity `json:"entities"`
	}

	MessageEntity struct {
		Type   string `json:"type"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
		URL    string `json:"url"`
		User   *User  `json:"user"`
	}

	Update struct {
		ID                int      `json:"update_id"`
		Message           *Message `json:"message"`
		EditedMessage     *Message `json:"edited_message"`
		ChannelPost       *Message `json:"channel_post"`
		EditedChannelPost *Message `json:"edited_message_post"`
	}

	ChatMember struct {
		User   User   `json:"user"`
		Status string `json:"status"`
	}
)
