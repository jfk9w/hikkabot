package telegram

import (
	"net/http"
)

const Endpoint = "https://api.telegram.org"

type (
	BotAPI interface {
		Stop()
		Me() *User
		In() <-chan Update
		GetMe() (*User, error)
		SendMessage(SendMessageRequest, bool, ResponseHandler)
		SendMessageSync(SendMessageRequest, bool) (*Response, error)
		SetChatTitle(ChatRef, string) (*bool, error)
		GetChat(ChatRef) (*Chat, error)
		GetChatAdministrators(ChatRef) ([]ChatMember, error)
		GetChatMember(ChatRef, UserID) (*ChatMember, error)
	}
)

func New(httpc *http.Client, token string,
	updates GetUpdatesRequest) (BotAPI, error) {
	ctx := &context{httpc, token}
	b := &impl{
		ctx: ctx,
	}

	_, err := b.GetMe()
	if err != nil {
		return nil, err
	}

	b.in, b.hs[0] = incoming(ctx, updates)
	b.outQ, b.outU, b.hs[1] = outgoing(ctx)

	return b, nil
}
