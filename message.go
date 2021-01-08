package newbee

import (
	"github.com/smartwalle/net4go"
	"sync"
)

type message struct {
	Type     messageType
	PlayerId uint64
	Packet   net4go.Packet
	Conn     net4go.Conn
}

type messageType int

const (
	messageTypeDefault   messageType = 0
	messageTypePlayerIn  messageType = 1
	messageTypePlayerOut messageType = 2
	messageTypeTick      messageType = 3
)

var messagePool = &sync.Pool{
	New: func() interface{} {
		return &message{}
	},
}

func newMessage(playerId uint64, mType messageType, packet net4go.Packet) *message {
	var m = messagePool.Get().(*message)
	m.PlayerId = playerId
	m.Type = mType
	m.Packet = packet
	return m
}

func releaseMessage(m *message) {
	if m != nil {
		m.PlayerId = 0
		m.Type = 0
		m.Packet = nil
		m.Conn = nil
		messagePool.Put(m)
	}
}

type messageQueue struct {
	items []*message
	mu    sync.Mutex
	cond  *sync.Cond
}

func (this *messageQueue) Enqueue(m *message) {
	this.mu.Lock()
	this.items = append(this.items, m)
	this.mu.Unlock()

	this.cond.Signal()
}

func (this *messageQueue) Reset() {
	this.items = this.items[0:0]
}

func (this *messageQueue) Dequeue(items *[]*message) {
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

func newQueue() *messageQueue {
	var q = &messageQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}
