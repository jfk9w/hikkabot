package util

type Handle struct {
	C  chan UnitType
	bC chan UnitType
}

func MultiHandle(hs ...Handle) Handle {
	h := NewHandle()
	go func() {
		defer h.Reply()
		<-h.C
		for _, h0 := range hs {
			h0.Ping()
		}
	}
	
	return h
}

func NewHandle() Handle {
	return Handle{
		C:  make(chan UnitType, 1),
		bC: make(chan UnitType, 1),
	}
}

func (h Handle) Ping() {
	h.C <- Unit
	<-h.bC
}

func (h Handle) Reply() {
	h.bC <- Unit
}
