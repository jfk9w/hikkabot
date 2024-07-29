package telegram

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"

	"github.com/jfk9w-go/flu/logf"
)

// Command is a text bot command.
type Command struct {
	Chat            *Chat
	User            *User
	Message         *Message
	Key             string
	Payload         string
	Args            []string
	CallbackQueryID string
}

func (cmd *Command) init(username Username, value string) {
	cmd.Key = value

	space := strings.Index(value, " ")
	if space > 0 && len(value) > space+1 {
		cmd.Key = value[:space]
		cmd.Payload = trim(value[space+1:])
	}

	at := strings.Index(cmd.Key, "@")
	if at > 0 && len(cmd.Key) > at+1 && string(username) == cmd.Key[at+1:] {
		cmd.Key = cmd.Key[:at]
	}

	cmd.Key = trim(cmd.Key)
	cmd.Payload = trim(cmd.Payload)

	cmd.Args = make([]string, 0)
	if cmd.Payload == "" {
		return
	}

	reader := csv.NewReader(strings.NewReader(cmd.Payload))
	reader.Comma = ' '
	reader.TrimLeadingSpace = true
	args, err := reader.Read()
	if err != nil {
		log().Errorf(nil, "parse %s args: %v", cmd, err)
		return
	}

	cmd.Args = args
}

func (cmd *Command) Arg(i int) string {
	if len(cmd.Args) > i {
		return cmd.Args[i]
	}

	return ""
}

func (cmd *Command) Reply(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID != "" {
		uText := utf8string.NewString(text)
		if uText.RuneCount() > 200 {
			text = uText.Slice(0, 197) + "..."
		}

		return client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &AnswerOptions{Text: text})
	}

	_, err := client.Send(ctx, cmd.Chat.ID, Text{Text: text}, &SendOptions{ReplyToMessageID: cmd.Message.ID})
	return err
}

func (cmd *Command) ReplyCallback(ctx context.Context, client Client, text string) error {
	if cmd.CallbackQueryID == "" {
		return nil
	}

	return cmd.Reply(ctx, client, text)
}

func (cmd *Command) collectArgs() string {
	b := new(strings.Builder)
	w := csv.NewWriter(b)
	w.Comma = ' '
	_ = w.Write(cmd.Args)
	w.Flush()
	return trim(b.String())
}

func (cmd *Command) Start(ctx context.Context, client Client) error {
	if cmd.CallbackQueryID == "" {
		return nil
	}

	data := base64.URLEncoding.EncodeToString([]byte(cmd.Key + " " + cmd.collectArgs()))
	if len(data) > 64 {
		logf.Get(client).Errorf(ctx, "start params too long for [%s]", cmd)
		return errors.New("start params too long")
	}

	url := fmt.Sprintf("https://t.me/%s?start=%s", string(client.Username()), data)
	return client.AnswerCallbackQuery(ctx, cmd.CallbackQueryID, &AnswerOptions{URL: url})
}

type Button [3]string

func (b Button) StartCallbackURL(username string) string {
	return fmt.Sprintf("https://t.me/%s?start=%s", username, url.QueryEscape(b[1]+" "+b[2]))
}

func (cmd *Command) Button(text string) Button {
	return Button{text, cmd.Key, cmd.collectArgs()}
}

func (cmd *Command) String() string {
	return fmt.Sprintf("%s [%s] from %s @ %s", cmd.Key, cmd.Payload, cmd.User.ID, cmd.Chat.ID)
}

type CommandListener interface {
	OnCommand(ctx context.Context, client Client, cmd *Command) error
}

type CommandListenerFunc func(context.Context, Client, *Command) error

func (fun CommandListenerFunc) OnCommand(ctx context.Context, client Client, cmd *Command) error {
	return fun(ctx, client, cmd)
}

type CommandRegistry map[string]CommandListener

func (r CommandRegistry) Add(key string, listener CommandListener) CommandRegistry {
	if _, ok := r[key]; ok {
		log().Panicf(nil, "duplicate command handler for %s", key)
	}

	r[key] = listener
	return r
}

func (r CommandRegistry) AddFunc(key string, listener CommandListenerFunc) CommandRegistry {
	return r.Add(key, listener)
}

func (r CommandRegistry) OnCommand(ctx context.Context, client Client, cmd *Command) error {
	if listener, ok := r[cmd.Key]; ok {
		return listener.OnCommand(ctx, client, cmd)
	}

	return nil
}

func (r CommandRegistry) From(v any) error {
	value := reflect.ValueOf(v)
	elemType := value.Type()
	for {
		for i := 0; i < elemType.NumMethod(); i++ {
			method := elemType.Method(i)
			methodType := method.Type
			if unicode.IsUpper([]rune(method.Name)[0]) && methodType.NumIn() == 4 && methodType.NumOut() == 1 &&
				methodType.In(1).AssignableTo(reflect.TypeOf(new(context.Context)).Elem()) &&
				methodType.In(2).AssignableTo(reflect.TypeOf(new(Client)).Elem()) &&
				methodType.In(3).AssignableTo(reflect.TypeOf(new(Command))) &&
				methodType.Out(0).AssignableTo(reflect.TypeOf(new(error)).Elem()) {

				value := value
				if methodType.In(0).Kind() != reflect.Pointer && value.Kind() == reflect.Pointer {
					value = reflect.Indirect(value)
				}

				name := strings.ToLower(method.Name)
				if strings.HasSuffix(name, "_callback") {
					name = name[:len(name)-9]
				} else {
					name = "/" + name
				}

				handle := CommandListenerFunc(func(ctx context.Context, client Client, command *Command) error {
					err := method.Func.Call([]reflect.Value{
						value,
						reflect.ValueOf(ctx),
						reflect.ValueOf(client),
						reflect.ValueOf(command),
					})[0].Interface()
					if err != nil {
						return err.(error)
					}

					return nil
				})

				r[name] = handle
			}
		}

		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
			continue
		}

		return nil
	}
}
