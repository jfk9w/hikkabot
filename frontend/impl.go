package frontend

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
)

type T struct {
	bot  Bot
	dvch Dvach
	back Backend
}

func (front *T) run() {
	commands := map[string]func(command){
		"status":      front.status,
		"sub":         front.subscribe,
		"subscribe":   front.subscribe,
		"unsub":       front.unsubscribe,
		"unsubscribe": front.unsubscribe,
		"clear":       front.unsubscribeAll,
		"dump":        front.dump,
		"catalog":     front.catalog,
		"front":       front.catalog,
	}

	for update := range front.bot.UpdateChannel() {
		cmd := front.parseUpdate(update)
		if cmd.string == "" {
			continue
		}

		if f, ok := commands[cmd.string]; ok {
			log.Infof("%s", cmd)
			go f(cmd)
		}
	}
}

var postHashRegex = regexp.MustCompile(`#?([A-Za-z]+)(\d+)`)

func (front *T) parseThread(value string) (*dvach.ID, int, error) {
	thread, offset, err := dvach.ParseThread(value)
	if err != nil {
		groups := postHashRegex.FindSubmatch([]byte(value))
		if len(groups) == 3 {
			thread := &dvach.ID{
				Board: strings.ToLower(string(groups[1])),
				Num:   string(groups[2]),
			}

			post, err := front.dvch.Post(*thread)
			if err != nil {
				log.Warningf("Unable to load post %s: %s", html.Num(thread.Board, thread.Num), err)
				return nil, 0, err
			}

			if post.Parent != "0" {
				offset, _ = strconv.Atoi(thread.Num)
				thread.Num = post.Parent
			}

			return thread, offset, nil
		}

		return nil, 0, err
	}

	return thread, offset, nil
}

func (front *T) hashify(thread dvach.ID) (string, error) {
	post, err := front.dvch.Post(thread)
	if err != nil {
		return "", err
	}

	return html.Hash(post.Subject), nil
}

func (front *T) parseUpdate(update telegram.Update) command {
	if update.Message == nil {
		return emptyCommand
	}

	text := update.Message.Text
	if !misc.IsFirstRune(text, '/') {
		return emptyCommand
	}

	tokens := strings.Split(text, " ")
	if len(tokens[0]) <= 1 {
		return emptyCommand
	}

	cmd := strings.ToLower(tokens[0][1:])
	bot, err := front.bot.GetMe()
	if err == nil {
		name := bot.Username
		cmd = strings.Replace(cmd, "@"+name, "", 1)
	}

	return command{cmd, tokens[1:], front.bot, update.Message.Chat.Ref(), update.Message.From.Ref()}
}
