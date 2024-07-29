package telegram

import "context"

type Client interface {
	GetMe(ctx context.Context) (*User, error)
	ForwardMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *SendOptions) (ID, error)
	CopyMessage(ctx context.Context, chatID ChatID, ref MessageRef, options *CopyOptions) (ID, error)
	DeleteMessage(ctx context.Context, ref MessageRef) error
	EditMessageReplyMarkup(ctx context.Context, ref MessageRef, markup ReplyMarkup) (*Message, error)
	ExportChatInviteLink(ctx context.Context, chatID ChatID) (string, error)
	GetChat(ctx context.Context, chatID ChatID) (*Chat, error)
	GetChatAdministrators(ctx context.Context, chatID ChatID) ([]ChatMember, error)
	GetChatMemberCount(ctx context.Context, chatID ChatID) (int64, error)
	GetChatMember(ctx context.Context, chatID ChatID, userID ID) (*ChatMember, error)
	AnswerCallbackQuery(ctx context.Context, id string, options *AnswerOptions) error
	Send(ctx context.Context, chatID ChatID, item Sendable, options *SendOptions) (*Message, error)
	SendChatAction(ctx context.Context, chatID ChatID, action string) error
	SendMediaGroup(ctx context.Context, chatID ChatID, media []Media, options *SendOptions) ([]Message, error)
	SetMyCommands(ctx context.Context, scope *BotCommandScope, commands []BotCommand) error
	GetMyCommands(ctx context.Context, scope *BotCommandScope) ([]BotCommand, error)
	DeleteMyCommands(ctx context.Context, scope *BotCommandScope) error
	Ask(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error)
	Answer(ctx context.Context, message *Message) error
	Username() Username
}

func InlineKeyboard(rows ...[]Button) ReplyMarkup {
	keyboard := make([][]InlineKeyboardButton, len(rows))
	for i, row := range rows {
		keyboard[i] = make([]InlineKeyboardButton, len(row))
		for j, button := range row {
			keyboard[i][j] = InlineKeyboardButton{
				Text:         button[0],
				CallbackData: button[1] + " " + button[2],
			}
		}
	}
	return &InlineKeyboardMarkup{keyboard}
}
