package newbee

import (
	"github.com/smartwalle/net4go"
	"time"
)

type Player struct {
	id    uint64
	index uint16
	token string

	isOnline bool
	isReady  bool

	conn *net4go.Conn

	loadProgress      int32
	lastHeartbeatTime int64
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

func (this *Player) UpdateLoadProgress(p int32) {
	this.loadProgress = p
}

func (this *Player) GetLoadProgress() int32 {
	return this.loadProgress
}

func (this *Player) RefreshHeartbeatTime() {
	this.lastHeartbeatTime = time.Now().Unix()
}

func (this *Player) GetHeartbeatTime() int64 {
	return this.lastHeartbeatTime
}

func (this *Player) Online(c *net4go.Conn) {
	if this.conn != nil && this.conn != c {
		this.conn.Close()
	}

	if this.conn != c {
		this.isReady = false
	}

	this.conn = c
	this.isOnline = true

	this.RefreshHeartbeatTime()
}

func (this *Player) Offline() {
	this.conn = nil
	this.isOnline = false
	this.isReady = false
}

func (this *Player) Ready() {
	if this.conn != nil && this.isOnline {
		this.isReady = true
	}
}

func (this *Player) UnReady() {
	this.isReady = false
}

func (this *Player) IsOnline() bool {
	return this.conn != nil && this.isOnline
}

func (this *Player) IsReady() bool {
	return this.conn != nil && this.isReady
}

// Cleanup 清理玩家的游戏信息，但是不断开连接
func (this *Player) Cleanup() {
	this.isReady = false
	this.loadProgress = 0
	this.lastHeartbeatTime = 0
}

// Close 关闭该玩家的所有信息，同时会断开连接
func (this *Player) Close() error {
	if this.conn != nil {
		this.conn.Close()
	}
	this.conn = nil

	this.isOnline = false

	this.Cleanup()

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
