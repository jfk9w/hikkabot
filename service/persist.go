package service

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

const persistInterval = 10 * time.Minute

type persister struct {
	filename string
	halt     util.Hook
	done     util.Hook
}

func newPersister(filename string) *persister {
	return &persister{
		filename: filename,
		halt:     util.NewHook(),
		done:     util.NewHook(),
	}
}

func (p *persister) start() {
	go func() {
		ticker := time.NewTicker(persistInterval)
		defer func() {
			p.done.Send()
			ticker.Stop()
		}()

		for {
			select {
			case <-p.halt:
				return

			case <-ticker.C:
				p.persist()
			}
		}
	}()
}

func (p *persister) stop() {
	p.halt.Send()
	p.done.Wait()
}

func (p *persister) persist() {
	_mutex.Lock()
	defer _mutex.Unlock()

	data, err := json.Marshal(_subs)
	if err != nil {
		sawmill.Error("persist: marshalling error " + err.Error())
		return
	}

	err = ioutil.WriteFile(p.filename, data, 0600)
	if err != nil {
		sawmill.Error("persist: file error" + err.Error())
		return
	}

	sawmill.Notice("persist ok")
}

func (p *persister) init() {
	data, err := ioutil.ReadFile(p.filename)
	if err != nil {
		sawmill.Error("load: file error " + err.Error())
		_subs = make(map[SubscriberKey]*Subscriber)
		return
	}

	subs := make(map[SubscriberKey]*Subscriber)
	err = json.Unmarshal(data, &subs)
	if err != nil {
		sawmill.Error("load: unmarshalling error " + err.Error())
		_subs = make(map[SubscriberKey]*Subscriber)
		return
	}

	for _, sub := range subs {
		sub.init()
	}

	_subs = subs
}
