package state

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jfk9w/hikkabot/telegram"
	"github.com/phemmer/sawmill"
)

type Service struct {
	registry map[SubscriberKey]Subscriber
	mutex    *sync.Mutex
}

func InitService(raw []byte) Service {
	mutex := new(sync.Mutex)
	if raw != nil {
		registry := newRegistry()
		err := json.Unmarshal(raw, &registry)
		if err == nil {
			return Service{
				registry: registry,
				mutex:    mutex,
			}
		} else {
			sawmill.Error(fmt.Sprintf("Unable to load registry: %s", err.Error()))
		}
	}

	return Service{
		registry: newRegistry(),
		mutex:    mutex,
	}
}

func (svc *Service) Subscribe()

func newRegistry() map[SubscriberKey]Subscriber {
	return make(map[SubscriberKey]Subscriber)
}

func (svc *Service) MarshalJSON() ([]byte, error) {
	return json.Marshal(svc.registry)
}

func (svc *Service) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &svc.registry)
}
