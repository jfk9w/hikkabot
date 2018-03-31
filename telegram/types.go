package telegram

type (
	MessageID = int
	ChatID    = int64
	UserID    = int

	User struct {
		ID           UserID `json:"id"`
		IsBot        bool   `json:"is_bot"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Username     string `json:"username"`
		LanguageCode string `json:"language_code"`
	}

	Chat struct {
		ID                          ChatID `json:"id"`
		Type                        string `json:"type"`
		Title                       string `json:"title"`
		Username                    string `json:"username"`
		FirstName                   string `json:"first_name"`
		LastName                    string `json:"last_name"`
		AllMembersAreAdministrators bool   `json:"all_members_are_administrators"`
		Description                 string `json:"description"`
		InviteLink                  string `json:"invite_link"`
		StickerSetName              string `json:"sticker_set_name"`
		CanSetStickerSet            bool   `json:"can_set_sticker_set"`
	}

	Message struct {
		ID                   MessageID       `json:"message_id"`
		From                 *User           `json:"from"`
		Date                 int             `json:"date"`
		Chat                 Chat            `json:"chat"`
		ForwardFrom          *User           `json:"forward_from"`
		ForwardFromChat      *Chat           `json:"forward_from_chat"`
		ForwardFromMessageID int             `json:"forward_from_message_id"`
		ForwardSignature     string          `json:"forward_signature"`
		ForwardDate          int             `json:"forward_date"`
		ReplyToMessage       *Message        `json:"reply_to_message"`
		EditDate             int             `json:"edit_date"`
		AuthorSignature      string          `json:"author_signature"`
		Text                 string          `json:"text"`
		Entities             []MessageEntity `json:"entities"`
		CaptionEntities      []MessageEntity `json:"caption_entities"`
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
		User                  User   `json:"user"`
		Status                string `json:"status"`
		UntilDate             string `json:"until_date"`
		CanBeEdited           bool   `json:"can_be_edited"`
		CanChangeInfo         bool   `json:"can_change_info"`
		CanPostMessages       bool   `json:"can_post_messages"`
		CanEditMessages       bool   `json:"can_edit_messages"`
		CanDeleteMessages     bool   `json:"can_delete_messages"`
		CanInviteUsers        bool   `json:"can_invite_users"`
		CanRestrictMembers    bool   `json:"can_restrict_members"`
		CanPinMessages        bool   `json:"can_pin_messages"`
		CanPromoteMembers     bool   `json:"can_promote_members"`
		CanSendMessages       bool   `json:"can_send_messages"`
		CanSendMediaMessages  bool   `json:"can_send_media_messages"`
		CanSendOtherMessages  bool   `json:"can_send_other_messages"`
		CanAddWebPagePreviews bool   `json:"can_add_web_page_previews"`
	}
)
