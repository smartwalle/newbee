package newbee

import (
	"sync"
)

type queue struct {
	items []*message
	mu    sync.Mutex
	cond  *sync.Cond
}

func (this *queue) Enqueue(m *message) {
	this.mu.Lock()
	this.items = append(this.items, m)
	this.mu.Unlock()

	this.cond.Signal()
}

func (this *queue) Reset() {
	this.items = this.items[0:0]
}

func (this *queue) Dequeue(items *[]*message) {
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

func newQueue() *queue {
	var q = &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}
