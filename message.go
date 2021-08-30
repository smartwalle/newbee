package newbee

import (
	"github.com/smartwalle/net4go"
	"sync"
)

type message struct {
	Type     messageType
	PlayerId int64
	Packet   net4go.Packet
	Session  net4go.Session
}

type messageType int

const (
	mTypeDefault   messageType = 0
	mTypePlayerIn  messageType = 1
	mTypePlayerOut messageType = 2
	mTypeTick      messageType = 3
)

type iMessageQueue interface {
	Enqueue(m *message)

	Dequeue(items *[]*message)

	Reset()
}

type blockMessageQueue struct {
	items []*message
	mu    sync.Mutex
	cond  *sync.Cond
}

func (this *blockMessageQueue) Enqueue(m *message) {
	this.mu.Lock()
	this.items = append(this.items, m)
	this.mu.Unlock()

	this.cond.Signal()
}

func (this *blockMessageQueue) Reset() {
	this.items = this.items[0:0]
}

func (this *blockMessageQueue) Dequeue(items *[]*message) {
	this.mu.Lock()
	for len(this.items) == 0 {
		this.cond.Wait()
	}
	this.mu.Unlock()

	this.mu.Lock()
	for _, item := range this.items {
		*items = append(*items, item)
		if item == nil {
			break
		}
	}

	this.Reset()

	this.mu.Unlock()
}

func newBlockQueue() *blockMessageQueue {
	var q = &blockMessageQueue{}
	q.cond = sync.NewCond(&q.mu)
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

func (this *messageQueue) Reset() {
	this.items = this.items[0:0]
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

	this.Reset()
	this.mu.Unlock()
}

func newQueue() *messageQueue {
	var q = &messageQueue{}
	return q
}
