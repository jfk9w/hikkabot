package util

type UnitType = struct{}

var Unit UnitType

type Hook chan UnitType

func NewHook() Hook {
	return Hook(make(chan UnitType, 1))
}

func (s Hook) Send() {
	s <- Unit
}

func (s Hook) Wait() {
	<-s
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
