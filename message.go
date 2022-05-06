package newbee

import (
	"github.com/smartwalle/queue/block"
	"sync"
	"sync/atomic"
)

type message struct {
	Type     messageType
	PlayerId int64
	Player   Player
	Data     interface{}
	Error    error
	rError   chan<- error
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

	Dequeue(items *[]*message) bool

	Close()
}

type blockMessageQueue struct {
	bq block.Queue[*message]
}

func (this *blockMessageQueue) Enqueue(m *message) {
	this.bq.Enqueue(m)
}

func (this *blockMessageQueue) Dequeue(items *[]*message) bool {
	return this.bq.Dequeue(items)
}

func (this *blockMessageQueue) Close() {
	this.bq.Close()
}

func newBlockQueue() *blockMessageQueue {
	var q = &blockMessageQueue{}
	q.bq = block.New[*message]()
	return q
}

type messageQueue struct {
	elements []*message
	mu       sync.Mutex
	closed   int32
}

func (this *messageQueue) Enqueue(m *message) {
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}

	this.mu.Lock()

	n := len(this.elements)
	c := cap(this.elements)
	if n+1 > c {
		npq := make([]*message, n, c*2)
		copy(npq, this.elements)
		this.elements = npq
	}
	this.elements = this.elements[0 : n+1]
	this.elements[n] = m

	this.mu.Unlock()
}

func (this *messageQueue) Dequeue(elements *[]*message) bool {
	this.mu.Lock()
	for len(this.elements) == 0 {
		this.mu.Unlock()
		return atomic.LoadInt32(&this.closed) != 1
	}

	for _, item := range this.elements {
		*elements = append(*elements, item)
	}

	this.elements = this.elements[0:0]
	this.mu.Unlock()
	return atomic.LoadInt32(&this.closed) != 1
}

func (this *messageQueue) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
	}
}

func newQueue() *messageQueue {
	var q = &messageQueue{}
	q.elements = make([]*message, 0, 32)
	return q
}
