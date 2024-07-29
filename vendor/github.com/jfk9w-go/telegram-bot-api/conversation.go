package telegram

import (
	"context"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/pkg/errors"
)

var (
	ErrUnexpectedAnswer = errors.New("unexpected answer")
)

type Question chan *Message

type Sender interface {
	Send(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error)
}

type conversationAware struct {
	sender    Sender
	questions map[ID]Question
	mu        syncf.RWMutex
}

func conversations(sender Sender) *conversationAware {
	return &conversationAware{
		sender:    sender,
		questions: make(map[ID]Question),
	}
}

func (a *conversationAware) Ask(ctx context.Context, chatID ChatID, sendable Sendable, options *SendOptions) (*Message, error) {
	if options == nil {
		options = new(SendOptions)
	}

	options.ReplyMarkup = ForceReply{ForceReply: true, Selective: true}
	m, err := a.sender.Send(ctx, chatID, sendable, options)
	if err != nil {
		return nil, errors.Wrap(err, "send question")
	}

	question, err := a.addQuestion(ctx, m.ID)
	if err != nil {
		return nil, errors.Wrap(err, "add question")
	}

	defer a.removeQuestion(ctx, m.ID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case answer := <-question:
		return answer, nil
	}
}

func (a *conversationAware) Answer(ctx context.Context, message *Message) error {
	ctx, cancel := a.mu.RLock(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	defer cancel()
	question, ok := a.questions[message.ReplyToMessage.ID]
	if !ok {
		return ErrUnexpectedAnswer
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case question <- message:
		return nil
	}
}

func (a *conversationAware) addQuestion(ctx context.Context, id ID) (Question, error) {
	question := make(Question)
	ctx, cancel := a.mu.Lock(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	defer cancel()

	if a.questions == nil {
		a.questions = make(map[ID]Question)
	}

	a.questions[id] = question
	return question, nil
}

func (a *conversationAware) removeQuestion(ctx context.Context, id ID) {
	ctx, cancel := a.mu.Lock(ctx)
	if ctx.Err() != nil {
		return
	}

	defer cancel()
	delete(a.questions, id)
}
