package util

type UnitType = struct{}

var Unit UnitType

type Signal chan UnitType

func NewSignal() Signal {
	return Signal(make(chan UnitType, 1))
}

func (s Signal) Read() <-chan UnitType {
	return chan UnitType(s)
}

func (s Signal) Send() {
	chan UnitType(s) <- Unit
}