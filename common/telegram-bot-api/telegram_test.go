package telegram

import (
	"os"
	"strconv"
	"testing"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

func TestT_Basic(t *testing.T) {
	var (
		token     = os.Getenv("TOKEN")
		target, _ = ParseChatID(os.Getenv("CHAT"))
		id        = Username("@self")
		config    = Config{
			APIConfig: APIConfig{
				Token:   token,
				Aliases: map[Username]ChatID{id: target},
				Http: &httpx.Config{
					Transport: &httpx.TransportConfig{
						Log: "httpx",
					},
				},
			},
			RouterConfig: DefaultIntervals,
		}
	)

	api := Configure(config, nil)
	_, err := api.SendMessage(id, "Please send a message in", nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 3; i > 0; i-- {
		_, err := api.SendMessage(id, strconv.Itoa(i), nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	var text string
	for update := range api.Updates {
		if update.Message.Chat.ID == target {
			text = update.Message.Text
			api.Updater.Close()
			break
		}
	}

	_, err = api.SendMessage(id, "You sent: "+text, nil)
	if err != nil {
		t.Fatal(err)
	}
}
