package keeper

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Offsets map[telegram.ChatRef]map[dvach.Ref]int

	T interface {
		SetOffset(telegram.ChatRef, dvach.Ref, int)
		DeleteOffset(telegram.ChatRef, dvach.Ref)
		GetOffsets() Offsets
	}
)

var log = logrus.GetLogger("keeper")
