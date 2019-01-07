package common

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Envelope struct {
	entity   interface{}
	sendOpts telegram.SendOpts
}

func (e Envelope) Send(b *telegram.Bot, chatID telegram.ID) error {
	var _, err = b.Send(chatID, e.entity, e.sendOpts)
	return err
}

func TextEnvelope(text string, disableWebPagePreview bool) Envelope {
	return Envelope{
		entity: text,
		sendOpts: telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Message().
			DisableWebPagePreview(disableWebPagePreview),
	}
}

func PhotoEnvelope(photo interface{}, caption string) Envelope {
	return Envelope{
		entity: photo,
		sendOpts: telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Media().
			Caption(caption).
			Photo(),
	}
}

func VideoEnvelope(video interface{}, caption string) Envelope {
	return Envelope{
		entity: video,
		sendOpts: telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Media().
			Caption(caption).
			Video(),
	}
}
