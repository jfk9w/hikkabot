package html2md

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
)

const token = ""

var posts = []string{
	"<a href=\"/b/res/166461181.html#166516176\" class=\"post-reply-link\" data-thread=\"166461181\" data-num=\"166516176\">>>166516176</a><br>У тебя интернет сломался педрила?<br><span class=\"spoiler\"><a href=\"http:&#47;&#47;www.xvideos.com&#47;video18232659&#47;anastasia_ass_got_plugged\" target=\"_blank\" rel=\"nofollow noopener noreferrer\">http:&#47;&#47;www.xvideos.com&#47;video18232659&#47;anastasia_ass_got_plugged</a></span>",
}

func TestParse(t *testing.T) {
	client := new(http.Client)
	bot := telegram.NewBotAPI(client, token)
	bot.Start()
	for _, post := range posts {
		msgs, _ := Parse(dvach.Post{
			Comment: post,
		})

		for _, msg := range msgs {
			done := util.NewHook()
			var err0 error
			bot.SendMessage(telegram.SendMessageRequest{
				Chat: telegram.ChatRef{
					ID: -1001181465085,
				},
				Text:      msg,
				ParseMode: telegram.Markdown,
			}, func(resp *telegram.Response, err error) {
				if err != nil {
					err0 = err
				} else if !resp.Ok {
					err0 = errors.New(resp.Description)
				}

				done.Send()
			}, true)

			done.Wait()
			if err0 != nil {
				t.Fatal(err0)
			}
		}
	}
}
