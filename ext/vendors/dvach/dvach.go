package dvach

import (
	"hikkabot/3rdparty/dvach"
	"hikkabot/core"
)

type Context interface {
	dvach.Context
	core.MediatorContext
}
