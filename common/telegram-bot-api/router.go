package telegram

import (
	"time"

	"github.com/jfk9w-go/hikkabot/common/gox/jsonx"
	"github.com/jfk9w-go/hikkabot/common/gox/serialx"
	"github.com/jfk9w-go/hikkabot/common/gox/syncx"
	"github.com/jfk9w-go/hikkabot/common/telegram-bot-api/api"
)

type RouterConfig struct {
	GatewayInterval    jsonx.Duration `json:"gateway_interval"`
	PrivateInterval    jsonx.Duration `json:"private_interval"`
	GroupInterval      jsonx.Duration `json:"group_interval"`
	SupergroupInterval jsonx.Duration `json:"supergroup_interval"`
	ChannelInterval    jsonx.Duration `json:"channel_interval"`
}

var DefaultIntervals = RouterConfig{
	GatewayInterval:    jsonx.Duration(30 * time.Millisecond),
	PrivateInterval:    jsonx.Duration(1 * time.Second),
	GroupInterval:      jsonx.Duration(3 * time.Second),
	SupergroupInterval: jsonx.Duration(3 * time.Second),
	ChannelInterval:    jsonx.Duration(0),
}

type action func() (*Message, error)

type routerOutput struct {
	message *Message
	err     error
}

func (ro *routerOutput) Status() serialx.Status {
	switch ro.err.(type) {
	case *TooManyMessages:
		return serialx.Delay
	case nil:
		return serialx.Ok
	default:
		return serialx.Failed
	}
}

func (ro *routerOutput) Delay() time.Duration {
	if err, ok := ro.err.(*TooManyMessages); ok {
		return err.RetryAfter
	}

	return 0
}

// Router is a wrapper around *API.
// Throttles outgoing messages and media.
type Router struct {
	*API
	RouterConfig

	chats, routes syncx.Map
	gateway       *serialx.T
}

func Route(api *API, intervals RouterConfig) *Router {
	gateway := serialx.New(intervals.GatewayInterval.Duration(), 3, 50)
	return &Router{
		API:          api,
		RouterConfig: intervals,
		chats:        syncx.NewMap(),
		routes:       syncx.NewMap(),
		gateway:      gateway,
	}
}

func (router *Router) route(ref Ref, action action) (*Message, error) {
	var (
		any   interface{}
		chat  *Chat
		route *serialx.T
		out   *routerOutput
		err   error
	)

	chat, err = router.GetChat(ref)
	if err != nil {
		return nil, err
	}

	any, _ = router.routes.ComputeIfAbsentExclusive(chat.ID, func() (interface{}, error) {
		var delay time.Duration
		switch chat.Type {
		case PrivateChatType:
			delay = router.RouterConfig.PrivateInterval.Duration()
		case GroupType:
			delay = router.RouterConfig.GroupInterval.Duration()
		case SupergroupType:
			delay = router.RouterConfig.SupergroupInterval.Duration()
		case ChannelType:
			delay = router.RouterConfig.ChannelInterval.Duration()
		}

		return serialx.New(delay, 0, 10), nil
	})

	route = any.(*serialx.T)
	out = route.Submit(func(_ interface{}) serialx.Out {
		return router.gateway.Submit(func(_ interface{}) serialx.Out {
			resp, err := action()
			return &routerOutput{
				message: resp,
				err:     err,
			}
		})
	}).(*routerOutput)

	return out.message, out.err
}

func (router *Router) SendMessage(id Ref, text string, opts *MessageOpts) (*Message, error) {
	return router.route(id, func() (*api.Message, error) {
		return router.API.SendMessage(id, text, opts)
	})
}

func (router *Router) SendPhoto(id Ref, media interface{}, opts *MediaOpts) (*api.Message, error) {
	return router.route(id, func() (*api.Message, error) {
		return router.API.SendPhoto(id, media, opts)
	})
}

func (router *Router) SendVideo(id api.Ref, media interface{}, opts *api.VideoOpts) (*api.Message, error) {
	return router.route(id, func() (*api.Message, error) {
		return router.API.SendVideo(id, media, opts)
	})
}

func (router *Router) GetChat(ref Ref) (*Chat, error) {
	var (
		any  interface{}
		chat *Chat
		err  error
	)

	any, err = router.chats.ComputeIfAbsentExclusive(ref, func() (interface{}, error) {
		return router.API.GetChat(ref)
	})

	if err != nil {
		return nil, err
	}

	chat = any.(*Chat)
	router.chats.Put(ref, chat)
	router.chats.Put(chat.ID, chat)
	if chat.Username.IsDefined() {
		router.chats.Put(chat.Username, chat)
	}

	return chat, nil
}

func (router *Router) Deroute(ref Ref) error {
	var (
		any   interface{}
		chat  *Chat
		route *serialx.T
		ok    bool
		err   error
	)

	chat, err = router.GetChat(ref)
	if err != nil {
		return err
	}

	any, ok = router.routes.Delete(chat.ID)
	if ok {
		route = any.(*serialx.T)
		route.Close()
	}

	return nil
}
