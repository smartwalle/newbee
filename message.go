package newbee

import (
	"sync"
)

type message struct {
	Type     messageType
	PlayerId int64
	Data     interface{}
	Error    error
}

type messageType int

const (
	mTypeDefault   messageType = 0
	mTypePlayerIn  messageType = 1
	mTypePlayerOut messageType = 2
	mTypeTick      messageType = 3
	mTypeCustom    messageType = 4
)

type iMessageQueue interface {
	Enqueue(m *message)

	Dequeue(items *[]*message)
}

type blockMessageQueue struct {
	items []*message
	cond  *sync.Cond
}

func (this *blockMessageQueue) Enqueue(m *message) {
	this.cond.L.Lock()
	this.items = append(this.items, m)
	this.cond.L.Unlock()

	this.cond.Signal()
}

func (this *blockMessageQueue) Dequeue(items *[]*message) {
	this.cond.L.Lock()
	for len(this.items) == 0 {
		this.cond.Wait()
	}

	for _, item := range this.items {
		*items = append(*items, item)
		if item == nil {
			break
		}
	}

	this.items = this.items[0:0]
	this.cond.L.Unlock()
}

func newBlockQueue() *blockMessageQueue {
	var q = &blockMessageQueue{}
	q.cond = sync.NewCond(&sync.Mutex{})
	return q
}

type messageQueue struct {
	items []*message
	mu    sync.Mutex
}

func (this *messageQueue) Enqueue(m *message) {
	this.mu.Lock()
	this.items = append(this.items, m)
	this.mu.Unlock()
}

func (this *messageQueue) Dequeue(items *[]*message) {
	this.mu.Lock()
	for len(this.items) == 0 {
		this.mu.Unlock()
		return
	}

	for _, item := range this.items {
		*items = append(*items, item)
		if item == nil {
			break
		}
	}

	this.items = this.items[0:0]
	this.mu.Unlock()
}

func newQueue() *messageQueue {
	var q = &messageQueue{}
	return q
}
