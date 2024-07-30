package dvach

import (
	"github.com/jfk9w/hikkabot/v4/internal/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/v4/internal/core"
)

type Context interface {
	dvach.Context
	core.MediatorContext
}
