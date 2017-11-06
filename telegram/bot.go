package telegram

import (
	"net/http"
	"strconv"
)

const Endpoint = "https://api.telegram.org"

type BotAPI struct {
	Me *User

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

func (svc *BotAPI) GetUpdatesChan(updates GetUpdatesRequest) <-chan Update {
	if svc.updates == nil {
		svc.updates = newUpdates(svc.gateway, updates)
		svc.updates.start()
	}

	return svc.updates.c
}

func (svc *BotAPI) Stop(choke bool) {
	if svc.updates != nil {
		<-svc.updates.stop()
	}

	<-svc.gateway.stop(choke)
	return
}

func (svc *BotAPI) MakeRequest(r Request) (*Response, error) {
	return svc.gateway.makeRequest(r)
}

// A simple method for testing your bot's auth token.
// Requires no parameters. Returns basic information about the bot in form of a User object.
func (svc *BotAPI) GetMe() (*User, error) {
	resp, err := svc.MakeRequest(GenericRequest{
		method: "getMe",
	})

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

// Use this method to send text messages. On success, the sent Message is returned.
func (svc *BotAPI) SendMessage(r SendMessageRequest, handler ResponseHandler, urgent bool) {
	svc.gateway.submit(r, handler, urgent)
}

// Use this method to change the title of a chat. Titles can't be changed for private chats.
// The bot must be an administrator in the chat for this to work
// and must have the appropriate admin rights. Returns True on success.
//
// Note: In regular groups (non-supergroups),
// this method will only work if the ‘All Members Are Admins’ setting is off in the target group.
//
// title - New chat title, 1-255 characters
func (svc *BotAPI) SetChatTitle(chat ChatRef, title string) (*bool, error) {
	resp, err := svc.MakeRequest(GenericRequest{
		method: "setChatTitle",
		params: map[string]string{
			"chat_id": chat.Key(),
			"title":   title,
		},
	})

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

// Use this method to get up to date information about the chat
// (current name of the user for one-on-one conversations, current username
// of a user, group or channel, etc.). Returns a Chat object on success.
func (svc *BotAPI) GetChat(chat ChatRef) (*Chat, error) {
	resp, err := svc.MakeRequest(GenericRequest{
		method: "getChat",
		params: map[string]string{
			"chat_id": chat.Key(),
		},
	})

	if err != nil {
		return nil, err
	}

	result := new(Chat)
	err = resp.Parse(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Use this method to get a list of administrators in a chat.
// On success, returns an Array of ChatMember objects that contains information
// about all chat administrators except other bots. If the chat is a group
// or a supergroup and no administrators were appointed, only the creator will be returned.
func (svc *BotAPI) GetChatAdministrators(chat ChatRef) ([]ChatMember, error) {
	resp, err := svc.MakeRequest(GenericRequest{
		method: "getChatAdministrators",
		params: map[string]string{
			"chat_id": chat.Key(),
		},
	})

	if err != nil {
		return nil, err
	}

	result := make([]ChatMember, 0)
	err = resp.Parse(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Use this method to get information about a member of a chat.
// Returns a ChatMember object on success.
func (svc *BotAPI) GetChatMembers(chat ChatRef, user UserID) (*ChatMember, error) {
	resp, err := svc.MakeRequest(GenericRequest{
		method: "getChatMember",
		params: map[string]string{
			"chat_id": chat.Key(),
			"user_id": strconv.Itoa(int(user)),
		},
	})

	if err != nil {
		return nil, err
	}

	result := new(ChatMember)
	err = resp.Parse(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
