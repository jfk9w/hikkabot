package main

import (
	"gopkg.in/telegram-bot-api.v4"
	"github.com/jfk9w/tele2ch/dvach"
	"time"
	"github.com/jfk9w/tele2ch/html2md"
	"fmt"
)

func main() {
	bot, _ := tgbotapi.NewBotAPI("")
	dv := dvach.NewAPI(dvach.APIConfig{ThreadFeedTimeout: 5 * time.Second})

	feed, _ := dv.GetThreadFeed("https://2ch.hk/b/res/162535733.html", 0)
	feed.Start()

	for i := 0; i < 50; i++ {
		select {
		case err := <-feed.Err:
			panic(err)

		case post := <-feed.C:
			msgs := html2md.Parse(post)
			for _, msg := range msgs {
				fmt.Println(msg)
				mc := tgbotapi.MessageConfig{
					BaseChat: tgbotapi.BaseChat{
						ChatID: 50613409,
					},
					ParseMode: tgbotapi.ModeMarkdown,
					Text: msg,
				}

				_, err := bot.Send(mc)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}
