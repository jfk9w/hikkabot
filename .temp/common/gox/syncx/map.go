package syncx

import (
	"sync"
)

type Entry struct {
	Key, Value interface{}
}

type Map struct {
	sync.RWMutex
	kv       map[interface{}]interface{}
	internal sync.Mutex
}

func NewMap() Map {
	return Map{
		RWMutex:  sync.RWMutex{},
		kv:       make(map[interface{}]interface{}),
		internal: sync.Mutex{},
	}
}

func (m *Map) Get(key interface{}) (interface{}, bool) {
	m.RLock()
	value, ok := m.kv[key]
	m.RUnlock()
	return value, ok
}

func (m *Map) Put(key interface{}, value interface{}) {
	m.Lock()
	m.kv[key] = value
	m.Unlock()
}

func (m *Map) PutIfAbsent(key interface{}, value interface{}) bool {
	m.Lock()
	if _, ok := m.kv[key]; ok {
		m.Unlock()
		return false
	}

	m.kv[key] = value
	m.Unlock()
	return true
}

func (m *Map) Update(key interface{}, f func(interface{}) interface{}) (interface{}, bool) {
	m.Lock()
	value, ok := m.kv[key]
	if !ok {
		m.Unlock()
		return nil, false
	}

	updated := f(value)
	m.kv[key] = updated
	m.Unlock()

	return updated, true
}

func (m *Map) UpdateAlways(key interface{}, f func(interface{}) interface{}) (interface{}, bool) {
	m.Lock()
	value := m.kv[key]
	updated := f(value)
	m.kv[key] = updated
	m.Unlock()

	return updated, true
}

func (m *Map) ComputeIfAbsent(key interface{}, f func() interface{}) (interface{}, bool) {
	value, ok := m.Get(key)
	if ok {
		return value, false
	}

	value = f()
	ok = m.PutIfAbsent(key, value)
	return value, ok
}

func (m *Map) ComputeIfAbsentExclusive(key interface{}, f func() (interface{}, error)) (interface{}, error) {
	m.internal.Lock()
	value, ok := m.Get(key)
	if ok {
		m.internal.Unlock()
		return value, nil
	}

	value, err := f()
	if err != nil {
		m.internal.Unlock()
		return nil, err
	}

	m.Put(key, value)
	m.internal.Unlock()

	return value, nil
}

func (m *Map) Delete(key interface{}) (interface{}, bool) {
	m.Lock()
	value, ok := m.kv[key]
	delete(m.kv, key)
	m.Unlock()
	return value, ok
}

func (m *Map) Save(fk func(interface{}) string) map[string]interface{} {
	kv := make(map[string]interface{})
	m.RLock()
	for k, v := range m.kv {
		if fk == nil {
			kv[k.(string)] = v
		} else {
			kv[fk(k)] = v
		}
	}

	m.RUnlock()
	return kv
}

func (m *Map) Restore(kv map[string]interface{}, fk func(string) interface{}) {
	m.Lock()
	for k, v := range kv {
		if fk == nil {
			m.kv[k] = v
		} else {
			m.kv[fk(k)] = v
		}
	}

	m.Unlock()
}
