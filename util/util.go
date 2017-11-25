package util

type UnitType = struct{}

var Unit UnitType

type Hook chan UnitType

func NewHook() Hook {
	return Hook(make(chan UnitType, 1))
}

func (s Hook) Listen() <-chan UnitType {
	return chan UnitType(s)
}

func (s Hook) Trigger() {
	chan UnitType(s) <- Unit
}