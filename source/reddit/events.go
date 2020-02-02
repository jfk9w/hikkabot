package reddit

type events []Event

func (e events) Len() int {
	return len(e)
}

func (e events) Less(i, j int) bool {
	return e[i].Ups < e[j].Ups
}

func (e events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
