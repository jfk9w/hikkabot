package telegram

import (
	"net/http"
)

const Endpoint = "https://api.telegram.org"

type BotAPI struct {
	Me      *User

	updates *updates
	gateway *gateway
}

func NewBotAPI(client *http.Client, token string) *BotAPI {
	if client == nil {
		client = new(http.Client)
	}

	return &BotAPI{
		gateway: newGateway(client, token),
	}
}

func (svc *BotAPI) Start() {
	me, err := svc.GetMe()
	if err != nil {
		panic(err)
	}

	svc.Me = me
	svc.gateway.start()
}

func (svc *BotAPI) GetUpdates(updates GetUpdatesRequest) <-chan Update {
	svc.updates = newUpdates(svc.gateway, updates)
	svc.updates.start()
	return svc.updates.c
}

func (svc *BotAPI) Stop(choke bool) <-chan struct{} {
	if svc.updates != nil {
		<-svc.updates.stop()
	}

	return svc.gateway.stop(choke)
}

func (svc *BotAPI) GetMe() (*User, error) {
	resp, err := svc.MakeRequest(GetMeRequest{})
	if err != nil {
		return nil, err
	}

	result := new(User)
	err = resp.Parse(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (svc *BotAPI) SendMessage(r SendMessageRequest, handler ResponseHandler, urgent bool) {
	svc.gateway.submit(r, handler, urgent)
}

func (svc *BotAPI) SetChatTitle(r SetChatTitleRequest) (*bool, error) {
	resp, err := svc.MakeRequest(r)
	if err != nil {
		return nil, err
	}

	result := new(bool)
	err = resp.Parse(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (svc *BotAPI) MakeRequest(r Request) (*Response, error) {
	return svc.gateway.makeRequest(r)
}