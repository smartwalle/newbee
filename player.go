package newbee

import (
	"github.com/smartwalle/net4go"
	"time"
)

type Player interface {
	// GetId 获取玩家 id
	GetId() uint64

	// GetToken 获取玩家 token
	GetToken() string

	// GetIndex 获取玩家索引
	GetIndex() uint16

	// UpdateLoadProgress 更新加载进度
	UpdateLoadProgress(int32)

	// GetLoadingProgress 获取加载进度
	GetLoadingProgress() int32

	// RefreshHeartbeatTime 刷新心跳包时间
	RefreshHeartbeatTime()

	// GetHeartbeatTime 最后获取到心跳包的时间
	GetHeartbeatTime() int64

	// Online 将连接和玩家进行绑定
	Online(*net4go.Conn)

	// Offline 断开玩家连接
	Offline()

	// Ready 将玩家标记为已准备就绪状态
	Ready()

	// UnReady 将玩家标记为未准备就绪状态
	UnReady()

	// IsOnline 获取玩家在线状态
	IsOnline() bool

	// IsReady 获取玩家准备状态
	IsReady() bool

	// SendMessage 发送消息
	SendMessage(net4go.Packet)

	// Cleanup 清理玩家的游戏信息，但是不断开连接
	Cleanup()

	// Close 关闭该玩家的所有信息，同时会断开连接
	Close() error
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

func (this *player) SendMessage(p net4go.Packet) {
	if this.IsOnline() == false {
		return
	}

	if this.conn.AsyncWritePacket(p, 0) != nil {
		this.Close()
	}
}

func (this *player) Cleanup() {
	this.isReady = false
	this.loadingProgress = 0
	this.lastHeartbeatTime = 0
}

func (this *player) Close() error {
	if this.conn != nil {
		this.conn.SetHandler(nil)
		this.conn.Close()
	}
	this.conn = nil

	this.isOnline = false

	this.Cleanup()

	return nil
}
