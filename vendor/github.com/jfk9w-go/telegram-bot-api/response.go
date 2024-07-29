package telegram

import (
	"fmt"
	"io"
	"time"

	"github.com/jfk9w-go/flu"
)

// responseParameters contains information about why a request was unsuccessful.
// See https://core.telegram.org/bots/api#responseparameters
type responseParameters struct {
	MigrateToChatID ID  `json:"migrate_to_chat_id"`
	RetryAfter      int `json:"retry_after"`
}

// response is a generic Telegram Bot API response.
// See https://core.telegram.org/bots/api#making-requests
type response struct {
	Ok          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      interface{}         `json:"result"`
	Parameters  *responseParameters `json:"parameters"`
}

func newResponse(value interface{}) *response {
	return &response{
		Result: value,
	}
}

func (r *response) DecodeFrom(reader io.Reader) error {
	if err := flu.JSON(r).DecodeFrom(reader); err != nil {
		return err
	}

	if !r.Ok {
		if r.Parameters != nil && r.Parameters.RetryAfter > 0 {
			return TooManyMessages{time.Duration(r.Parameters.RetryAfter) * time.Second}
		}

		return Error{r.ErrorCode, r.Description}
	}

	return nil
}

// Error is an error returned by Telegram Bot API.
// See https://core.telegram.org/bots/api#making-requests
type Error struct {
	// ErrorCode is an integer error code.
	ErrorCode int
	// Description explains the error.
	Description string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d %s", e.ErrorCode, e.Description)
}

// TooManyMessages is returned in case of exceeding flood control.
type TooManyMessages struct {
	RetryAfter time.Duration
}

func (e TooManyMessages) Error() string {
	return fmt.Sprintf("too many messages, retry after %.0f seconds", e.RetryAfter.Seconds())
}
