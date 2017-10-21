package main

import (
	"io/ioutil"
	"json"
	"sync"

	"github.com/jfk9w/tele2ch/telegram"
)

type Sub struct {
	Mgmt       telegram.ChatRef `json:"mgmt"`
	ThreadLink string           `json:"thread_link"`
	Offset     int              `json:"offset"`
	stop       chan struct{}
}

type Subs struct {
	r     map[string][]Sub
	mutex *sync.Mutex
}

func NewSubs() *Subs {
	return &Subs{
		r:     make(map[string][]Sub),
		mutex: new(sync.Mutex),
	}
}

func LoadSubs(filename string) (*Subs, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	r := make(map[string][]Sub)
	err = json.Unmarshal(data, &subs)
	if err != nil {
		return nil, err
	}

	return &Subs{
		r:     r,
		mutex: new(sync.Mutex),
	}
}

func (subs *Subs) Register(chat telegram.ChatRef, sub Sub) {
	key := chat.Key()
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	if curr, ok := subs.r[key]; !ok {
		subs.r[key] = make([]Sub, 0)
	}

	subs.r[key] = append(subs.r[key], sub)
}

func (subs *Subs) Unregister(chat telegram.ChatRef) {
	key := chat.Key()
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	delete(subs.r, key)
}

func (subs *Subs) Save(filename string) error {
	subs.mutex.Lock()
	c := make(map[string]Sub)
	for k, v := range subs.r {
		c[k] = v
	}

	subs.mutex.Unlock()
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	return nil
}
