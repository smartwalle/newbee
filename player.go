package newbee

import (
	"github.com/smartwalle/net4go"
	"time"
)

type Player interface {
	GetId() uint64

	GetToken() string

	GetIndex() uint16

	UpdateLoadProgress(p int32)

	GetLoadingProgress() int32

	RefreshHeartbeatTime()

	GetHeartbeatTime() int64

	Online(c *net4go.Conn)

	Offline()

	Ready()

	UnReady()

	IsOnline() bool

	IsReady() bool

	Cleanup()

	Close() error

	SendMessage(p net4go.Packet)
}

type player struct {
	id    uint64
	index uint16
	token string

	isOnline bool
	isReady  bool

	conn *net4go.Conn

	loadingProgress   int32
	lastHeartbeatTime int64
}

func NewPlayer(id uint64, token string, index uint16) Player {
	var p = &player{}
	p.id = id
	p.token = token
	p.index = index
	return p
}

func (this *player) GetId() uint64 {
	return this.id
}

func (this *player) GetToken() string {
	return this.token
}

func (this *player) GetIndex() uint16 {
	return this.index
}

func (this *player) UpdateLoadProgress(p int32) {
	this.loadingProgress = p
}

func (this *player) GetLoadingProgress() int32 {
	return this.loadingProgress
}

func (this *player) RefreshHeartbeatTime() {
	this.lastHeartbeatTime = time.Now().Unix()
}

func (this *player) GetHeartbeatTime() int64 {
	return this.lastHeartbeatTime
}

func (this *player) Online(c *net4go.Conn) {
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

func (this *player) Offline() {
	this.conn = nil
	this.isOnline = false
	this.isReady = false
}

func (this *player) Ready() {
	if this.conn != nil && this.isOnline {
		this.isReady = true
	}
}

func (this *player) UnReady() {
	this.isReady = false
}

func (this *player) IsOnline() bool {
	return this.conn != nil && this.isOnline
}

func (this *player) IsReady() bool {
	return this.conn != nil && this.isReady
}

// Cleanup 清理玩家的游戏信息，但是不断开连接
func (this *player) Cleanup() {
	this.isReady = false
	this.loadingProgress = 0
	this.lastHeartbeatTime = 0
}

// Close 关闭该玩家的所有信息，同时会断开连接
func (this *player) Close() error {
	if this.conn != nil {
		this.conn.Close()
	}
	this.conn = nil

	this.isOnline = false

	this.Cleanup()

	return nil
}

func (this *player) SendMessage(p net4go.Packet) {
	if this.IsOnline() == false {
		return
	}

	if this.conn.AsyncWritePacket(p, 0) != nil {
		this.Close()
	}
}
