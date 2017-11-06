package service

import (
	"testing"
)

func TestBasicService(t *testing.T) {
	scenario := Compose(t,
		Event{
			source:  "instance",
			payload: "start",
			callback: func(svc *Service) {
				if !svc.handle().IsActive() {
					t.Fatal("Service is not active")
				}
			},
		},
		Event{
			source:  "instance",
			payload: "work",
			callback: func(svc *Service) {
				Stop(svc, true)
			},
		},
		Event{
			source:  "instance",
			payload: "stop",
		},
	)

	service := &Service{
		id:       "instance",
		scenario: scenario,
		_handle:  NewHandle(),
	}

	Start(service)
}

type Event struct {
	source   string
	payload  string
	callback func(svc *Service)
}

func (e Event) Check(t *testing.T, source string, payload string) {
	if e.source != source || e.payload != payload {
		t.Fatalf("Expected event: %s/%s, actual event: %s/%s",
			e.source, e.payload, source, payload)
	}
}

type Scenario struct {
	t       *testing.T
	steps   []Event
	current int
}

func Compose(t *testing.T, steps ...Event) *Scenario {
	return &Scenario{t, steps, 0}
}

func (s *Scenario) Step(svc *Service, payload string) {
	e := s.steps[s.current]
	e.Check(s.t, svc.id, payload)
	if e.callback != nil {
		e.callback(svc)
	}

	s.current++
}

func (s *Scenario) Attach(svc T) {
	svc.(*Service).scenario = s
	for _, dep := range svc.deps() {
		s.Attach(dep)
	}
}

type Service struct {
	id       string
	scenario *Scenario
	worked   bool
	_handle  *Handle
	_deps    []T
}

func (svc *Service) step(payload string) {
	svc.scenario.Step(svc, payload)
}

func (svc *Service) start() {
	svc.step("start")
}

func (svc *Service) work() {
	if !svc.worked {
		svc.step("work")
		svc.worked = true
	}
}

func (svc *Service) stop() {
	svc.step("stop")
}

func (svc *Service) handle() *Handle {
	return svc._handle
}

func (svc *Service) deps() []T {
	return svc._deps
}
