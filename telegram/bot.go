package telegram

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jfk9w/hikkabot/util"
)

const (
	Endpoint = "https://api.telegram.org"
)

type BotAPI struct {
	Me *User

	ctx      *context
	qcO, ucO chan<- DeferredRequest
	cI       <-chan Update

	hs [2]util.Handle
}

func NewBotAPIWithClient(client *http.Client,
	token string, updates GetUpdatesRequest) (*BotAPI, error) {

	ctx := &context{client, token}
	b := &BotAPI{
		ctx: ctx,
	}

	_, err := b.GetMe()
	if err != nil {
		return nil, err
	}

	cI, hI := incoming(ctx, updates)
	qcO, ucO, hO := outgoing(ctx)

	b.cI = cI
	b.qcO = qcO
	b.ucO = ucO
	b.hs = [2]util.Handle{hI, hO}

	return b, nil
}

func (b *BotAPI) request(req Request) (*Response, error) {
	return b.ctx.request(req)
}

func (b *BotAPI) GetUpdatesChan(updates GetUpdatesRequest) <-chan Update {
	return b.cI
}

func (b *BotAPI) Stop() {
	for _, h := range b.hs {
		h.Ping()
	}

	close(b.ucO)
	close(b.qcO)
}

func (b *BotAPI) GetMe() (*User, error) {
	resp, err := b.request(GenericRequest{
		method: "getMe",
	})

	if err != nil {
		return nil, err
	}

	user := new(User)
	err = resp.Parse(user)
	if err != nil {
		return nil, err
	}

	b.Me = user
	return user, nil
}

func (b *BotAPI) SendMessage(r SendMessageRequest, urgent bool, handler ResponseHandler) {
	req := DeferredRequest{r, handler}
	if urgent {
		b.ucO <- req
	} else {
		b.qcO <- req
	}
}

func (b *BotAPI) SendMessageSync(req SendMessageRequest, urgent bool) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	c := make(chan util.UnitType, 1)
	handler := func(resp0 *Response, err0 error) {
		resp = resp0
		err = err0
		c <- util.Unit
	}

	b.SendMessage(req, urgent, handler)
	<-c

	return resp, err
}

func (b *BotAPI) SetChatTitle(chat ChatRef, title string) (*bool, error) {
	resp, err := b.request(GenericRequest{
		method: "setChatTitle",
		params: map[string]string{
			"chat_id": chat.Key(),
			"title":   title,
		},
	})

	if err != nil {
		return nil, err
	}

	isOk := new(bool)
	err = resp.Parse(isOk)
	if err != nil {
		return nil, err
	}

	return isOk, nil
}

func (b *BotAPI) GetChat(ref ChatRef) (*Chat, error) {
	resp, err := b.request(GenericRequest{
		method: "getChat",
		params: map[string]string{
			"chat_id": ref.Key(),
		},
	})

	if err != nil {
		return nil, err
	}

	chat := new(Chat)
	err = resp.Parse(chat)
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (b *BotAPI) GetChatAdministrators(chat ChatRef) ([]ChatMember, error) {
	resp, err := b.request(GenericRequest{
		method: "getChatAdministrators",
		params: map[string]string{
			"chat_id": chat.Key(),
		},
	})

	if err != nil {
		return nil, err
	}

	cm := make([]ChatMember, 0)
	err = resp.Parse(&cm)
	if err != nil {
		return nil, err
	}

	return cm, nil
}

func (b *BotAPI) GetChatMembers(chat ChatRef, user UserID) (*ChatMember, error) {
	resp, err := b.request(GenericRequest{
		method: "getChatMember",
		params: map[string]string{
			"chat_id": chat.Key(),
			"user_id": strconv.Itoa(int(user)),
		},
	})

	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		return nil, fmt.Errorf("%d", resp.ErrorCode)
	}

	cm := new(ChatMember)
	err = resp.Parse(cm)
	if err != nil {
		return nil, err
	}

	return cm, nil
}
