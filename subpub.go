package main

import (
	"sync"
)

type DataChannel chan Payload

type Subscribers map[int]DataChannel

// EventBus stores the information about subscribers
type EventBus struct {
	subscribers Subscribers
	rm          sync.RWMutex
}

func (eb *EventBus) Publish(data Payload) {
	eb.rm.RLock()

	go func(data Payload, subs Subscribers) {
		for _, ch := range subs {
			ch <- data
		}
	}(data, eb.subscribers)

	eb.rm.RUnlock()
}

func (eb *EventBus) Subscribe(key int, ch DataChannel) {
	eb.rm.Lock()
	eb.subscribers[key] = ch
	eb.rm.Unlock()
}
func (eb *EventBus) Unsbscribe(key int) {
	eb.rm.Lock()
	defer eb.rm.Unlock()
	delete(eb.subscribers, key)
}
