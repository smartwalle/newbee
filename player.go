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
	GetId() uint64

	// GetToken 获取玩家 token
	GetToken() string

	// GetType 获取玩家所属的类型, 比如用于区分是可以正常进行游戏操作的玩家还是只能观战的观察者
	GetType() uint32

	// GetIndex 获取玩家索引
	GetIndex() uint32

	// Conn
	Conn() net4go.Conn

	// Connect 将连接和玩家进行绑定
	Connect(conn net4go.Conn)

	// Disconnect 断开玩家连接
	Disconnect()

	// Connected 获取玩家在线状态
	Connected() bool

	// SendMessage 发送消息
	SendMessage([]byte)

	// SendPacket 发送消息
	SendPacket(net4go.Packet)

	// Close 关闭该玩家的所有信息，同时会断开连接
	Close() error
}

type player struct {
	id    uint64
	pType uint32
	index uint32
	token string

	conn net4go.Conn
}

func NewPlayer(id uint64, opts ...PlayerOption) Player {
	var p = &player{}
	p.id = id
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (this *player) GetId() uint64 {
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

func (this *player) Conn() net4go.Conn {
	return this.conn
}

func (this *player) Connect(c net4go.Conn) {
	if this.conn != nil && this.conn != c {
		this.conn.Close()
	}

	this.conn = c
}

func (this *player) Disconnect() {
	if this.conn != nil {
		this.conn.Close()
	}
	this.conn = nil
}

func (this *player) Connected() bool {
	return this.conn != nil && this.conn.Closed() == false
}

func (this *player) SendMessage(b []byte) {
	if this.conn == nil {
		return
	}
	if err := this.conn.AsyncWrite(b, 0); err != nil {
		this.Close()
	}
}

func (this *player) SendPacket(p net4go.Packet) {
	if this.conn == nil {
		return
	}
	if err := this.conn.AsyncWritePacket(p, 0); err != nil {
		this.Close()
	}
}

func (this *player) Close() error {
	this.Disconnect()
	return nil
}
