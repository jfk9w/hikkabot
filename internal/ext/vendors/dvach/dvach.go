package dvach

import (
	"github.com/jfk9w/hikkabot/internal/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/internal/core"
)

type Context interface {
	dvach.Context
	core.MediatorContext
}
