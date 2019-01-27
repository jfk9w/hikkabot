package api

import (
	"os"
	"testing"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

func TestAPI_Basic(t *testing.T) {
	var (
		token     = os.Getenv("TOKEN")
		id        = Username("test")
		target, _ = ParseChatID(os.Getenv("CHAT"))
		config    = Config{
			Token:   token,
			Aliases: map[Username]ChatID{id: target},
		}

		api = New(config)
	)

	me, err := api.GetMe()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Me: %+v\n", me)

	chat, err := api.GetChat(id)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Chat: %+v\n", chat)

	msg, err := api.SendMessage(id, "Hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Message: %+v\n", msg)

	msg, err = api.SendPhoto(id, &httpx.File{Path: "testdata/check.png"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Message: %+v\n", msg)
}
