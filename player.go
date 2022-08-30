package newbee

import (
	"github.com/smartwalle/net4go"
)

type Player interface {
	// GetId 获取玩家 id
	GetId() int64

	// Session 获取连接信息
	Session() net4go.Session

	// Connected 获取玩家在线状态
	Connected() bool

	// SendPacket 发送消息
	SendPacket(net4go.Packet)

	// AsyncSendPacket 异步发送消息
	AsyncSendPacket(net4go.Packet)

	// Close 关闭玩家
	// 注意：不要重写本方法，如果需要清理玩家信息，应该在 Game 的 OnLeaveRoom 中完成
	Close() error
}

type player struct {
	sess net4go.Session
	id   int64
}

func NewPlayer(id int64, sess net4go.Session) Player {
	var p = &player{}
	p.id = id
	p.sess = sess
	return p
}

func (this *player) GetId() int64 {
	return this.id
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
