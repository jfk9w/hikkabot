package subscription

import (
	"fmt"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Key struct {
	ChatID telegram.ID
	Name   string
	Active bool
}

func (k *Key) String() string {
	var state string
	if k.Active {
		state = "active"
	} else {
		state = "inactive"
	}

	return fmt.Sprintf("%s:%s:%s", k.ChatID, k.Name, state)
}

var ErrInvalidKey = errors.New("invalid Key")

func (k *Key) Parse(str string) error {
	parts := strings.Split(str, ":")
	if len(parts) != 3 {
		return ErrInvalidKey
	}

	chatID, err := telegram.ParseID(parts[0])
	if err != nil {
		return errors.Wrap(err, "on parsing Telegram ID")
	}

	name := parts[1]
	active := false
	if parts[2] == "active" {
		active = true
	}

	k.ChatID = chatID
	k.Name = name
	k.Active = active

	return nil
}
