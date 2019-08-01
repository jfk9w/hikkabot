package subscription

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

// Item defines a subscription.
type Item interface {

	// Service should return a human-readable description of the service this subscription is for.
	Service() string

	// ID should return a subscription ID (will be compared only for the same chats).
	ID() string

	// Name should return a human-readable name of this subscription.
	Name() string

	// Parse should try to initialize a subscription from a given input string and an optional options string.
	Parse(ctx Context, cmd string, opts string) error

	// Update is called when a subscription is called for update.
	Update(ctx Context, offset Offset, uc *UpdateCollection)
}

type Service = func() Item

var ErrParseFailed = errors.New("failed to parse")

type itemData struct {
	Item
	PrimaryID   string
	SecondaryID string
	ChatID      telegram.ID
	Offset      Offset
}
