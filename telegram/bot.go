package telegram

import (
	"net/http"
)

const Endpoint = "https://api.telegram.org"

type BotAPI struct {
	Me      *User
	Updates *Updates
	gateway *Gateway
}

func NewBotAPI(client *http.Client, token string) *BotAPI {
	if client == nil {
		client = new(http.Client)
	}

	return &BotAPI{
		gateway: NewGateway(client, token),
	}
}

func (svc *BotAPI) Start(updates *GetUpdatesRequest) {
	me, err := svc.GetMe()
	if err != nil {
		panic(err)
	}

	svc.Me = me

	err = svc.gateway.Start()
	if err != nil {
		panic(err)
	}

	if updates != nil {
		svc.Updates = NewUpdates(svc.gateway, *updates)
		err = svc.Updates.Start()
		if err != nil {
			panic(err)
		}
	}
}

func (svc *BotAPI) Stop(choke bool) {
	if svc.Updates != nil {
		svc.Updates.Stop()
	}

	svc.gateway.Stop(choke)
}

func (svc *BotAPI) GetMe() (*User, error) {
	resp, err := svc.gateway.MakeRequest(GetMeRequest{})
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

func (svc *BotAPI) SendMessage(r SendMessageRequest, callback ResponseHandler) {
	svc.gateway.Submit(r, callback)
}

func (svc *BotAPI) SendUrgentMessage(r SendMessageRequest, callback ResponseHandler) {
	svc.gateway.Urgent(r, callback)
}

func (svc *BotAPI) SetChatTitle(r SetChatTitleRequest) (*bool, error) {
	resp, err := svc.gateway.MakeRequest(r)
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