package subscription

import (
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/pkg/errors"
)

type Channel interface {
	SendUpdate(telegram.ID, Update) error
}

type Telegram struct {
	telegram.Client
}

func (tg *Telegram) SendUpdate(chatID telegram.ID, update Update) error {
	parseMode := update.Text.ParseMode
	pages := update.Text.Pages
	if parseMode != telegram.HTML {
		panic(errors.Errorf("unsupported parse mode: %s", parseMode))
	}
	if len(update.Media) == 1 && len(pages) == 1 {
		media := update.Media[0]
		mediaURL := format.PrintHTMLLink("[media]", media.URL())
		caption := mediaURL + "\n" + pages[0]
		if utf8.RuneCountInString(caption) <= telegram.MaxCaptionSize {
			res, typ, err := media.Wait()
			if err == nil {
				_, err = tg.Send(chatID,
					&telegram.Media{
						Type:      typ,
						Readable:  res,
						Caption:   caption,
						ParseMode: parseMode},
					&telegram.SendOptions{
						DisableNotification: true})
			}
			if err != nil {
				_, err = tg.Send(chatID,
					&telegram.Text{
						Text:      caption,
						ParseMode: parseMode},
					&telegram.SendOptions{
						DisableNotification: true})
			}
			return err
		}
	}

	for _, page := range pages {
		_, err := tg.Send(chatID,
			&telegram.Text{
				Text:                  page,
				ParseMode:             parseMode,
				DisableWebPagePreview: true},
			&telegram.SendOptions{
				DisableNotification: true})
		if err != nil {
			return err
		}
	}

	for _, media := range update.Media {
		url := format.PrintHTMLLink("[media]", media.URL())
		res, typ, err := media.Wait()
		if err == nil {
			_, err = tg.Send(chatID,
				&telegram.Media{
					Type:      typ,
					Readable:  res,
					Caption:   url,
					ParseMode: parseMode},
				&telegram.SendOptions{
					DisableNotification: true})
		}
		if err != nil {
			_, err = tg.Send(chatID,
				&telegram.Text{
					Text:      url,
					ParseMode: parseMode},
				&telegram.SendOptions{
					DisableNotification: true})
		}
		if err != nil {
			return err
		}
	}

	return nil
}
