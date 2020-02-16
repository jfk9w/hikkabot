package metrics

var Dummy Metrics = dummy{}

type dummy struct {
}

func (dummy) WithPrefix(prefix string) Metrics {
	return Dummy
}

func (dummy) Counter(name string, labels Labels) Counter {
	return DummyCounter
}

func (dummy) Gauge(name string, labels Labels) Gauge {
	return DummyGauge
}

func (dummy) Close() error {
	return nil
}

var DummyCounter Counter = dummyCounter{}

type dummyCounter struct {
}

func (dummyCounter) Inc() {

}

func (dummyCounter) Add(f float64) {

}

var DummyGauge Gauge = dummyGauge{}

type dummyGauge struct {
}

func (dummyGauge) Set(f float64) {

}

func (dummyGauge) Inc() {

}

func (dummyGauge) Dec() {

}

func (dummyGauge) Add(f float64) {

}

func (dummyGauge) Sub(f float64) {

}
