package frontend

import (
	"regexp"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	bot  Bot
	dvch Dvach
	back Backend
}

func (front *T) Run() {
	commands := map[string]func(Command){
		"status":      front.status,
		"sub":         front.subscribe,
		"subscribe":   front.subscribe,
		"unsub":       front.unsubscribe,
		"unsubscribe": front.unsubscribe,
		"clear":       front.unsubscribeAll,
		"dump":        front.dump,
		"catalog":     front.search,
		"search":      front.search,
	}

	for update := range front.bot.UpdateChannel() {
		cmd := front.ParseUpdate(update)
		if cmd.string == "" {
			continue
		}

		if f, ok := commands[cmd.string]; ok {
			log.Infof("%s", cmd)
			go f(cmd)
		}
	}
}

var postHashtagRegex = regexp.MustCompile(`#?([A-Za-z]+)(\d+)`)

func (front *T) ParseHashtag(value string) (ref dvach.Ref, offset int, err error) {
	groups := postHashtagRegex.FindSubmatch([]byte(value))
	if len(groups) == 3 {
		if ref, err = dvach.ToRef(string(groups[1]), string(groups[2])); err != nil {
			return
		}

		var post *dvach.Post
		post, err = front.dvch.Post(ref)
		if err != nil {
			log.Warningf("Unable to load post %s: %s", ref, err)
			return
		}

		if post.Parent != 0 {
			offset = ref.Num

			ref.NumString = post.ParentString
			ref.Num = post.Parent
		}

		return ref, offset, nil
	}

	return dvach.Ref{}, 0, errors.Errorf("invalid value: %s", value)
}

func (front *T) ParseThread(value string) (ref dvach.Ref, offset int, err error) {
	ref, offset, err = dvach.ParseUrl(value)
	if err != nil {
		ref, offset, err = front.ParseHashtag(value)
	}

	return
}

func (front *T) Hashtag(ref dvach.Ref) (text.Hashtag, error) {
	op, err := front.dvch.Thread(ref)
	if err != nil {
		return "", err
	}

	return text.FormatSubject(op.Subject), nil
}

func (front *T) ParseUpdate(update telegram.Update) Command {
	var message *telegram.Message
	if update.Message != nil {
		message = update.Message
	} else if update.EditedMessage != nil {
		message = update.EditedMessage
	} else {
		return emptyCommand
	}

	text := message.Text
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

	return Command{cmd, tokens[1:], front.bot, message.Chat.Ref(), message.From.Ref()}
}
