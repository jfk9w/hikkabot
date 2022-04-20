package iface

import "github.com/pkg/errors"

var (
	ErrInvalidHeader = errors.New("invalid header")

	errSubscribe = errors.New("" +
		"Usage: /subscribe SUB [CHAT_ID] [OPTIONS]\n\n" +
		"SUB – subscription string (for example, a link).\n" +
		"CHAT_ID – target chat username or '.' to use this chat. Optional, this chat by default.\n" +
		"OPTIONS – subscription-specific options string. Optional, empty by default.")

	errDeleteAll = errors.New("" +
		"Usage: /clear PATTERN [CHAT_ID]\n\n" +
		"PATTERN – pattern to match subscription error.\n" +
		"CHAT_ID – target chat username or '.' to use this chat.",
	)
)
