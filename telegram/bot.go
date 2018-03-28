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

func New(httpc *http.Client, tokens []string,
	updates GetUpdatesRequest) (BotAPI, error) {
	tokenQ := make(chan string, len(tokens))
	for _, token := range tokens {
		tokenQ <- token
	}

	ctx := &context{httpc, tokenQ}
	b := &impl{
		ctx: ctx,
	}

	_, err := b.GetMe()
	if err != nil {
		return nil, err
	}

	b.in, b.hs[0] = incoming(ctx, updates)
	b.outQ, b.outU, b.hs[1] = outgoing(ctx, len(tokens))

	return b, nil
}
