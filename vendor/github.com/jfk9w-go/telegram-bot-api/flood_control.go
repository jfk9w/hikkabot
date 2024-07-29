package telegram

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

// GatewaySendDelay is a delay between two consecutive /send* API calls per bot token.
var GatewaySendDelay = 35 * time.Millisecond

// SendDelays are delays between two consecutive /send* API calls per chat with a given type.
var SendDelays = map[ChatType]time.Duration{
	PrivateChat: 35 * time.Millisecond,
	GroupChat:   3 * time.Second,
	Supergroup:  time.Second,
	Channel:     3 * time.Second,
}

var MaxSendRetries = 3

type executor interface {
	Execute(ctx context.Context, method string, body flu.EncoderTo, resp interface{}) error
}

type floodControlAware struct {
	clock    syncf.Clock
	executor executor
	lock     syncf.Locker
	locks    map[ChatID]syncf.Locker
	once     sync.Once
	mu       syncf.RWMutex
}

var errUnknownRecipient = errors.New("unknown recipient")

func (c *floodControlAware) send(ctx context.Context, chatID ChatID, item sendable, options *SendOptions, resp interface{}) error {
	c.once.Do(func() {
		c.lock = syncf.Semaphore(c.clock, 1, GatewaySendDelay)
	})

	body, err := options.body(chatID, item)
	if err != nil {
		return errors.Wrap(err, "failed to write send data")
	}

	method := "send" + strings.Title(item.kind())
	lock, ok := c.getLock(chatID)
	if ok {
		ctx, cancel := lock.Lock(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		defer cancel()
	}

	ctx, cancel := c.lock.Lock(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	defer cancel()

	for i := 0; i <= MaxSendRetries; i++ {
		err = c.executor.Execute(ctx, method, body, resp)
		var timeout time.Duration
		switch err := err.(type) {
		case nil:
			if ok {
				return nil
			} else {
				return errUnknownRecipient
			}
		case TooManyMessages:
			logf.Get(c).Warnf(ctx, "too many messages, sleeping for %s", err.RetryAfter)
			timeout = err.RetryAfter
		case Error:
			return err
		default:
			timeout = GatewaySendDelay
		}

		if err := flu.Sleep(ctx, timeout); err != nil {
			return err
		}
	}

	return err
}

func (c *floodControlAware) getLock(chatID ChatID) (syncf.Locker, bool) {
	_, cancel := c.mu.RLock(nil)
	defer cancel()
	lock, ok := c.locks[chatID]
	return lock, ok
}

func (c *floodControlAware) createLock(chat *Chat) {
	_, cancel := c.mu.Lock(nil)
	defer cancel()

	if _, ok := c.locks[chat.ID]; ok {
		return
	}

	if c.locks == nil {
		c.locks = make(map[ChatID]syncf.Locker)
	}

	lock := syncf.Semaphore(c.clock, 1, chat.Type.SendDelay())
	c.locks[chat.ID] = lock
	if chat.Username != nil {
		c.locks[*chat.Username] = lock
	}
}

// Send is an umbrella method for various /send* API calls which return only one Message.
// See
//   https://core.telegram.org/bots/api#sendmessage
//   https://core.telegram.org/bots/api#sendphoto
//   https://core.telegram.org/bots/api#sendvideo
//   https://core.telegram.org/bots/api#senddocument
//   https://core.telegram.org/bots/api#sendaudio
//   https://core.telegram.org/bots/api#sendvoice
//   https://core.telegram.org/bots/api#sendsticker
func (c *floodControlAware) Send(ctx context.Context, chatID ChatID, item Sendable, options *SendOptions) (*Message, error) {
	m := new(Message)
	err := c.send(ctx, chatID, item, options, m)
	if err == errUnknownRecipient {
		c.createLock(&m.Chat)
		err = nil
	}

	return m, err
}

// SendMediaGroup is used to send a group of photos or videos as an album.
// On success, an array of Message's is returned.
// See https://core.telegram.org/bots/api#sendmediagroup
func (c *floodControlAware) SendMediaGroup(ctx context.Context, chatID ChatID, media []Media, options *SendOptions) ([]Message, error) {
	ms := make([]Message, 0)
	err := c.send(ctx, chatID, MediaGroup(media), options, &ms)
	if err == errUnknownRecipient {
		c.createLock(&ms[0].Chat)
		err = nil
	}

	return ms, err
}
