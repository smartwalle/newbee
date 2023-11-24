package newbee

import (
	"github.com/smartwalle/queue/block"
	"sync"
	"sync/atomic"
)

type message struct {
	Player   Player
	Data     interface{}
	Error    error
	rError   chan<- error
	Type     messageType
	PlayerId int64
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

func (q *blockMessageQueue) Enqueue(m *message) {
	q.bq.Enqueue(m)
}

func (q *blockMessageQueue) Dequeue(items *[]*message) bool {
	return q.bq.Dequeue(items)
}

func (q *blockMessageQueue) Close() {
	q.bq.Close()
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

func (q *messageQueue) Enqueue(m *message) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return
	}

	q.mu.Lock()

	n := len(q.elements)
	c := cap(q.elements)
	if n+1 > c {
		npq := make([]*message, n, c*2)
		copy(npq, q.elements)
		q.elements = npq
	}
	q.elements = q.elements[0 : n+1]
	q.elements[n] = m

	q.mu.Unlock()
}

func (q *messageQueue) Dequeue(elements *[]*message) bool {
	q.mu.Lock()
	for len(q.elements) == 0 {
		q.mu.Unlock()
		return atomic.LoadInt32(&q.closed) != 1
	}

	for _, item := range q.elements {
		*elements = append(*elements, item)
	}

	q.elements = q.elements[0:0]
	q.mu.Unlock()
	return atomic.LoadInt32(&q.closed) != 1
}

func (q *messageQueue) Close() {
	if atomic.CompareAndSwapInt32(&q.closed, 0, 1) {
	}
}

func newQueue() *messageQueue {
	var q = &messageQueue{}
	q.elements = make([]*message, 0, 32)
	return q
}
