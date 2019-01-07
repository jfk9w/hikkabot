package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/segmentio/ksuid"

	. "github.com/jfk9w-go/telegram-bot-api"
)

type Offset = int64
type RawOptions = json.RawMessage

type BaseSubscribeService struct {
	storage   Storage
	scheduler *Scheduler
	b         *Bot
	Type      ServiceType
}

func BaseSubscribe(storage Storage, scheduler *Scheduler, b *Bot) BaseSubscribeService {
	return BaseSubscribeService{
		storage:   storage,
		scheduler: scheduler,
		b:         b,
	}
}

func (svc BaseSubscribeService) ServiceType() ServiceType {
	return svc.Type
}

func (svc BaseSubscribeService) Suspend(id string, err error) {
	subscription := svc.storage.Suspend(id, err)
	go svc.notifyAdministrators(subscription.ChatID, subscription.Name,
		"Subscription suspended.",
		fmt.Sprintf("Reason: %s", err))
}

func (svc BaseSubscribeService) readOptions(rawOptions RawOptions, options interface{}) error {
	return json.Unmarshal(rawOptions, options)
}

func (svc BaseSubscribeService) writeOptions(options interface{}) (RawOptions, error) {
	return json.Marshal(options)
}

func (svc BaseSubscribeService) subscribe(chatId ID, name string, secondaryId string, options interface{}) error {
	rawOptions, err := svc.writeOptions(options)
	if err != nil {
		return err
	}

	if ok := svc.storage.Insert(&Subscription{
		ID:          ksuid.New().String(),
		SecondaryID: secondaryId,
		ChatID:      chatId,
		Type:        svc.Type,
		Name:        name,
		Options:     rawOptions,
	}); !ok {
		return errors.New("exists")
	}

	go svc.notifyAdministrators(chatId, name, "Subscription OK.", "")
	svc.scheduler.Schedule(chatId)

	return nil
}

func (svc BaseSubscribeService) notifyAdministrators(chatId ID, subscription, header, footer string) {
	chat, err := svc.b.GetChat(chatId)
	if err != nil {
		log.Printf("Failed to get chat %d: %s", chatId, err)
		return
	}

	var (
		chatTitle string
		adminIds  []ID
	)

	if chat.Type == PrivateChat {
		chatTitle = "private"
		adminIds = []ID{chat.ID}
	} else {
		chatTitle = chat.Title
		admins, err := svc.b.GetChatAdministrators(chatId)
		if err != nil {
			log.Printf("Failed to get administrator list for chat %d: %s", chatId, err)
			return
		}

		adminIds = make([]ID, 0)
		for _, admin := range admins {
			if admin.User.IsBot {
				continue
			}

			adminIds = append(adminIds, admin.User.ID)
		}
	}

	text := header + fmt.Sprintf(`
Chat: %s
Service: %s
Subscription: %s
`, chatTitle, svc.Type, subscription) + footer

	for _, adminId := range adminIds {
		_, err := svc.b.Send(adminId, text, NewSendOpts().
			Message().
			DisableWebPagePreview(true))
		if err != nil {
			log.Printf("Failed to send message to %d: %s", adminId, err)
		}
	}
}
