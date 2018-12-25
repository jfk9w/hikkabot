package api

import (
	"encoding/json"
	"fmt"
	"time"
)

type responseParameters struct {
	MigrateToChatID ChatID `json:"migrate_to_chat_id"`
	RetryAfter      int    `json:"retry_after"`
}

type response struct {
	Ok          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      json.RawMessage     `json:"result"`
	Parameters  *responseParameters `json:"parameters"`
}

func (resp *response) parse(target interface{}) error {
	if !resp.Ok {
		if resp.Parameters != nil && resp.Parameters.RetryAfter > 0 {
			return &TooManyMessages{time.Duration(resp.Parameters.RetryAfter) * time.Second}
		}

		return &Error{resp.ErrorCode, resp.Description}
	}

	if target == nil {
		return nil
	}

	data, err := resp.Result.MarshalJSON()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

type Error struct {
	ErrorCode   int
	Description string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s", e.ErrorCode, e.Description)
}

type TooManyMessages struct {
	RetryAfter time.Duration
}

func (e *TooManyMessages) Error() string {
	return fmt.Sprintf("too many messages, retry after %.0f seconds", e.RetryAfter.Seconds())
}
