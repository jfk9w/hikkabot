package keeper

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
)

type (
	Config struct {
		DBPath      string  `json:"db_path"`
		SyncTimeout *int    `json:"sync_timeout"`
		Logger      *string `json:"logger"`
	}

	Offsets map[telegram.ChatRef]map[dvach.Ref]int

	T interface {
		SetOffset(telegram.ChatRef, dvach.Ref, int)
		DeleteOffset(telegram.ChatRef, dvach.Ref)
		GetOffsets() Offsets
	}
)
