package telegram

import (
	"fmt"
	"github.com/jfk9w/hikkabot/util"
	"strconv"
)

type impl struct {
	me *User

	ctx        *context
	outQ, outU chan<- DeferredRequest
	in         <-chan Update

	hs [2]util.Handle
}

func (b *impl) request(req Request) (*Response, error) {
	return b.ctx.request(req)
}

func (b *impl) Stop() {
	for _, h := range b.hs {
		h.Ping()
	}

	close(b.outU)
	close(b.outQ)
}

func (b *impl) Me() *User {
	return b.me
}

func (b *impl) In() <-chan Update {
	return b.in
}

func (b *impl) GetMe() (*User, error) {
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

	b.me = user
	return user, nil
}

func (b *impl) SendMessage(r SendMessageRequest, urgent bool, handler ResponseHandler) {
	req := DeferredRequest{r, handler}
	if urgent {
		b.outU <- req
	} else {
		b.outQ <- req
	}
}

func (b *impl) SendMessageSync(req SendMessageRequest, urgent bool) (*Response, error) {
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

func (b *impl) SetChatTitle(chat ChatRef, title string) (*bool, error) {
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

func (b *impl) GetChat(ref ChatRef) (*Chat, error) {
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

func (b *impl) GetChatAdministrators(chat ChatRef) ([]ChatMember, error) {
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

func (b *impl) GetChatMember(chat ChatRef, user UserID) (*ChatMember, error) {
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
