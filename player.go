package newbee

import (
	"github.com/smartwalle/net4go"
	"sync/atomic"
)

type Player struct {
	id    uint64
	index uint16
	token string

	isOnline int32
	isReady  int32

	conn *net4go.Conn
}

func NewPlayer(id uint64, token string, index uint16) *Player {
	var p = &Player{}
	p.id = id
	p.token = token
	p.index = index

	return p
}

func (this *Player) GetId() uint64 {
	return this.id
}

func (this *Player) GetToken() string {
	return this.token
}

func (this *Player) GetIndex() uint16 {
	return this.index
}

func (this *Player) Connect(c *net4go.Conn) {
	this.conn = c
	atomic.StoreInt32(&this.isOnline, 1)
	atomic.StoreInt32(&this.isReady, 1)
}

func (this *Player) IsOnline() bool {
	return this.conn != nil && atomic.LoadInt32(&this.isOnline) == 1
}

func (this *Player) IsReady() bool {
	return this.conn != nil && atomic.LoadInt32(&this.isReady) == 1
}

func (this *Player) Close() error {
	if this.conn != nil {
		this.conn.Close()
	}
	this.conn = nil

	atomic.StoreInt32(&this.isOnline, 0)
	atomic.StoreInt32(&this.isReady, 0)

	return nil
}

func (this *Player) SendMessage(p net4go.Packet) {
	if this.IsOnline() == false {
		return
	}

	if this.conn.AsyncWritePacket(p, 0) != nil {
		this.Close()
	}
}
