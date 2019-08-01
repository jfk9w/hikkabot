package subscription

import (
	"strings"
)

type event interface {
	status() string
	details(*strings.Builder)
	undo() string
}

var resume resumeEvent

type resumeEvent struct {
}

func (e resumeEvent) status() string {
	return "OK 🔥"
}

func (e resumeEvent) details(sb *strings.Builder) {

}

func (e resumeEvent) undo() string {
	return "suspend"
}

type suspendEvent struct {
	err error
}

func (e *suspendEvent) status() string {
	return "suspended ⏸"
}

func (e *suspendEvent) details(sb *strings.Builder) {
	sb.WriteString("\nReason: ")
	sb.WriteString(e.err.Error())
}

func (e *suspendEvent) undo() string {
	return "resume"
}
