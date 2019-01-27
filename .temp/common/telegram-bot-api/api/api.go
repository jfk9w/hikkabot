package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

const StatusFlood = 429

type T struct {
	httpx.HTTP
	token   string
	aliases map[Ref]Ref
}

func New(config Config) *T {
	if config.Token == "" {
		panic("empty token")
	}

	var api = &T{
		HTTP: httpx.Configure(
			config.Http.WithStatusCodes(
				http.StatusOK,
				http.StatusSeeOther,
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				StatusFlood,
				http.StatusTooManyRequests,
				http.StatusInternalServerError,
			)),
		token:   config.Token,
		aliases: make(map[Ref]Ref),
	}

	for alias, ref := range config.Aliases {
		api.aliases[alias] = ref
	}

	return api
}

func (api *T) SendVideo(id Ref, video interface{}, opts *VideoOpts) (*Message, error) {
	return api.sendMedia(id, "video", video, opts.params())
}

func (api *T) SendPhoto(id Ref, photo interface{}, opts *MediaOpts) (*Message, error) {
	return api.sendMedia(id, "photo", photo, opts.params())
}

func (api *T) SendDocument(id Ref, document interface{}, opts *MediaOpts) (*Message, error) {
	return api.sendMedia(id, "document", document, opts.params())
}

func (api *T) sendMedia(id Ref, mediaType string, media interface{}, params params) (*Message, error) {
	var (
		method = "send" + strings.Title(mediaType)
		req    = &request{method, params.Add(
			"chat_id", api.unalias(id).Value())}
		resp = new(Message)
		err  error
	)

	switch media := media.(type) {
	case string:
		req.Params.Add(mediaType, media)
		err = api.exec(req, resp, nil)

	case *httpx.File:
		err = api.exec(req, resp, httpx.Multipart{mediaType: media})

	default:
		panic(fmt.Sprintf("invalid media type: %T", media))
	}

	return resp, err
}

func (api *T) SendMessage(id Ref, text string, opts *MessageOpts) (*Message, error) {
	var (
		req = &request{"sendMessage", opts.params().Add(
			"chat_id", api.unalias(id).Value()).Add(
			"text", text)}

		resp = new(Message)
	)

	return resp, api.exec(req, resp, nil)
}

func (api *T) GetUpdates(opts *UpdatesOpts) ([]Update, error) {
	var (
		req  = &request{"getUpdates", opts.params()}
		resp = make([]Update, 0)
	)

	return resp, api.exec(req, &resp, nil)
}

func (api *T) GetMe() (*User, error) {
	var (
		req  = &request{"getMe", params{}}
		resp = new(User)
	)

	return resp, api.exec(req, resp, nil)
}

func (api *T) GetChat(id Ref) (*Chat, error) {
	var (
		req = &request{"getChat", params{
			"chat_id": {api.unalias(id).Value()},
		}}

		resp = new(Chat)
	)

	return resp, api.exec(req, resp, nil)
}

func (api *T) GetChatAdministrators(id Ref) ([]ChatMember, error) {
	var (
		req = &request{"getChatAdministrators", params{
			"chat_id": {api.unalias(id).Value()},
		}}

		resp = make([]ChatMember, 0)
	)

	return resp, api.exec(req, &resp, nil)
}

func (api *T) GetChatMember(id Ref, user ChatID) (*ChatMember, error) {
	var (
		req = &request{"getChatMember", params{
			"chat_id": {api.unalias(id).Value()},
			"user_id": {user.Value()},
		}}

		resp = new(ChatMember)
	)

	return resp, api.exec(req, resp, nil)
}

func (api *T) exec(req *request, target interface{}, multipart httpx.Multipart) (err error) {
	var (
		url    = api.url(req.Method)
		resp   = new(response)
		output = &httpx.JSON{
			Value: resp,
		}
	)

	if multipart == nil {
		err = api.Post(url, req.Params, output)
	} else {
		err = api.Multipart(url, req.Params, multipart, output)
	}

	if err != nil {
		return err
	}

	if target != nil {
		err = resp.parse(target)
	}

	return
}

func (api *T) url(method string) string {
	return "https://api.telegram.org/bot" + api.token + "/" + method
}

func (api *T) unalias(id Ref) Ref {
	if a, ok := api.aliases[id]; ok {
		return a
	}

	return id
}
