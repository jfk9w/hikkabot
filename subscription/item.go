package subscription

import (
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

// Item defines a subscription.
type Item interface {

	// Service should return a human-readable description of the service this subscription is for.
	Service() string

	// ID should return a subscription ID (will be compared only for the same chats).
	ID() string

	// name should return a human-readable name of this subscription.
	Name() string

	// Parse should try to initialize a subscription from a given input string and an optional options string.
	Parse(ctx Context, cmd string, opts string) error

	// Update is called when a subscription is called for update.
	Update(ctx Context, offset int64, session *UpdateQueue)
}

type Service = func() Item

var ErrParseFailed = errors.New("failed to parse")

type ItemData struct {
	Item
	PrimaryID   string
	SecondaryID string
	ChatID      telegram.ID
	Offset      int64
}

func (item *ItemData) String() string {
	return fmt.Sprintf("%v (%s / %v @ %v)", item.PrimaryID, item.Service(), item.ID(), item.ChatID)
}
