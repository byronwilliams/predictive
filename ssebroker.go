package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type SSEBroker struct {
	ConnectedClients map[chan []byte]bool
	locker           *sync.RWMutex
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		ConnectedClients: make(map[chan []byte]bool),
		locker:           &sync.RWMutex{},
	}
}

func (sb *SSEBroker) Monitor() {
	for {
		sb.locker.RLock()
		log.Println("Num Clients", len(sb.ConnectedClients))
		sb.locker.RUnlock()
		time.Sleep(time.Second * 15)
	}
}

func (sb *SSEBroker) AddClient(ch chan []byte) {
	sb.locker.Lock()
	sb.ConnectedClients[ch] = true
	sb.locker.Unlock()
}

func (sb *SSEBroker) RemoveClient(ch chan []byte) {
	sb.locker.Lock()
	delete(sb.ConnectedClients, ch)
	sb.locker.Unlock()
}

func (sb *SSEBroker) NewReading(r reading) {
	j, _ := json.Marshal(r)
	sb.locker.RLock()
	for cl := range sb.ConnectedClients {
		go func(client chan []byte) {
			client <- j
		}(cl)
	}
	sb.locker.RUnlock()
}
