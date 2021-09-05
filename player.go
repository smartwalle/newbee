package newbee

import (
	"github.com/smartwalle/net4go"
)

type PlayerOption func(*player)

func WithPlayerToken(token string) PlayerOption {
	return func(p *player) {
		p.token = token
	}
}

func WithPlayerType(pType uint32) PlayerOption {
	return func(p *player) {
		p.pType = pType
	}
}

func WithPlayerIndex(index uint32) PlayerOption {
	return func(p *player) {
		p.index = index
	}
}

type Player interface {
	// GetId 获取玩家 id
	GetId() int64

	// GetToken 获取玩家 token
	GetToken() string

	// GetType 获取玩家所属的类型, 比如用于区分是可以正常进行游戏操作的玩家还是只能观战的观察者
	GetType() uint32

	// GetIndex 获取玩家索引
	GetIndex() uint32

	// Session 获取连接信息
	Session() net4go.Session

	// Connected 获取玩家在线状态
	Connected() bool

	// SendPacket 发送消息
	SendPacket(net4go.Packet)

	// AsyncSendPacket 异步发送消息
	AsyncSendPacket(net4go.Packet)

	// Close 关闭该玩家的所有信息，同时会断开连接
	Close() error
}

type player struct {
	id    int64
	pType uint32
	index uint32
	token string

	sess net4go.Session
}

func NewPlayer(id int64, sess net4go.Session, opts ...PlayerOption) Player {
	var p = &player{}
	p.id = id
	for _, opt := range opts {
		opt(p)
	}
	p.sess = sess
	return p
}

func (this *player) GetId() int64 {
	return this.id
}

func (this *player) GetToken() string {
	return this.token
}

func (this *player) GetType() uint32 {
	return this.pType
}

func (this *player) GetIndex() uint32 {
	return this.index
}

func (this *player) Session() net4go.Session {
	return this.sess
}

func (this *player) Connected() bool {
	return this.sess != nil && this.sess.Closed() == false
}

func (this *player) SendPacket(p net4go.Packet) {
	if this.sess == nil {
		return
	}
	if err := this.sess.WritePacket(p); err != nil {
		this.Close()
	}
}

func (this *player) AsyncSendPacket(p net4go.Packet) {
	if this.sess == nil {
		return
	}
	if err := this.sess.AsyncWritePacket(p); err != nil {
		this.Close()
	}
}

func (this *player) Close() error {
	if this.sess != nil {
		this.sess.Close()
	}
	this.sess = nil
	return nil
}
