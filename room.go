package newbee

import (
	"errors"
	"github.com/smartwalle/net4go"
	"sync"
	"sync/atomic"
	"time"
)

const (
	kPlayerId = "player_id"
)

type message struct {
	PlayerId uint64
	Packet   net4go.Packet
}

var (
	ErrRoomClosed  = errors.New("newbee: room is closed")
	ErrRoomRunning = errors.New("newbee: room is running")
	ErrNilGame     = errors.New("newbee: game is nil")
)

const (
	kRoomStatePending = iota // 等待游戏运行
	kRoomStateRunning        // 有游戏在运行
	kRoomStateClose          // 房间已关闭
)

const (
	kDefaultMessageBuffer = 1024
	kDefaultPlayerBuffer  = 10
)

type roomOptions struct {
	MessageBuffer int
	PlayerBuffer  int
}

func newRoomOptions() *roomOptions {
	var o = &roomOptions{}
	o.MessageBuffer = kDefaultMessageBuffer
	o.PlayerBuffer = kDefaultPlayerBuffer
	return o
}

type RoomOption interface {
	Apply(*roomOptions)
}

type roomOptionFun func(options *roomOptions)

func (f roomOptionFun) Apply(o *roomOptions) {
	f(o)
}

func WithMessageBuffer(buffer int) RoomOption {
	return roomOptionFun(func(o *roomOptions) {
		if buffer <= 0 {
			buffer = kDefaultMessageBuffer
		}
		o.MessageBuffer = buffer
	})
}

func WithPlayerBuffer(buffer int) RoomOption {
	return roomOptionFun(func(o *roomOptions) {
		if buffer <= 0 {
			buffer = kDefaultPlayerBuffer
		}
		o.PlayerBuffer = buffer
	})
}

type Room interface {
	// GetId 获取房间 id
	GetId() uint64

	// GetPlayer 获取玩家信息
	GetPlayer(playerId uint64) Player

	// GetPlayers 获取所有的玩家信息
	GetPlayers() []Player

	// Connect 将玩家和连接进行绑定
	Connect(playerId uint64, conn *net4go.Conn)

	// JoinPlayer 加入新的玩家，如果连接不为空，则将该玩家和连接进行绑定
	JoinPlayer(player Player, conn *net4go.Conn)

	// RunGame 启动游戏
	RunGame(game Game, opts ...RoomOption) error

	// SendMessage 向指定玩家发送消息
	SendMessage(playerId uint64, packet net4go.Packet)

	// Broadcast 向房间中的所有玩家广播消息
	Broadcast(packet net4go.Packet)

	// CheckAllPlayerOnline 检测房间中的所有玩家是否都在线
	CheckAllPlayerOnline() bool

	// CheckAllPlayerReady 检测房间中的所有玩家是否都准备就绪
	CheckAllPlayerReady() bool

	// GetPlayerCount 获取房间中的玩家数量
	GetPlayerCount() int

	// GetOnlinePlayerCount 获取房间中在线玩家数量
	GetOnlinePlayerCount() int

	// GetReadyPlayerCount 获取房间中准备就绪玩家数量
	GetReadyPlayerCount() int

	// Close 关闭房间
	Close() error
}

// --------------------------------------------------------------------------------
type room struct {
	id      uint64
	state   uint32
	mu      sync.RWMutex
	players map[uint64]Player

	messageChan   chan *message
	playerInChan  chan *net4go.Conn
	playerOutChan chan *net4go.Conn

	closeChan chan struct{}
}

func NewRoom(roomId uint64, players []Player) Room {
	var r = &room{}
	r.id = roomId
	r.state = kRoomStatePending
	r.players = make(map[uint64]Player)
	for _, player := range players {
		r.players[player.GetId()] = player
	}
	return r
}

// --------------------------------------------------------------------------------
func (this *room) GetId() uint64 {
	return this.id
}

func (this *room) GetPlayer(playerId uint64) Player {
	this.mu.RLock()
	defer this.mu.RUnlock()

	if playerId == 0 {
		return nil
	}

	var p = this.players[playerId]
	return p
}

func (this *room) GetPlayers() []Player {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var ps = make([]Player, 0, len(this.players))
	for _, player := range this.players {
		ps = append(ps, player)
	}
	return ps
}

// --------------------------------------------------------------------------------
func (this *room) Connect(playerId uint64, c *net4go.Conn) {
	if c != nil {
		c.Set(kPlayerId, playerId)
		c.SetHandler(this)

		select {
		case this.playerInChan <- c:
		default:
		}
	}
}

func (this *room) JoinPlayer(player Player, c *net4go.Conn) {
	if player != nil {
		this.mu.Lock()
		defer this.mu.Unlock()

		// 玩家不存在则添加该玩家
		if _, ok := this.players[player.GetId()]; ok == false {
			this.players[player.GetId()] = player
		}

		if c != nil {
			this.Connect(player.GetId(), c)
		}
	}
}

// --------------------------------------------------------------------------------
func (this *room) RunGame(game Game, opts ...RoomOption) error {
	if game == nil {
		return ErrNilGame
	}

	if atomic.LoadUint32(&this.state) == kRoomStateClose {
		return ErrRoomClosed
	}

	if atomic.LoadUint32(&this.state) == kRoomStateRunning {
		return ErrRoomRunning
	}

	atomic.StoreUint32(&this.state, kRoomStateRunning)

	defer func() {
		game.OnCloseRoom()
		this.Close()
	}()

	var options = newRoomOptions()
	for _, o := range opts {
		o.Apply(options)
	}
	this.messageChan = make(chan *message, options.MessageBuffer)
	this.playerInChan = make(chan *net4go.Conn, options.PlayerBuffer)
	this.playerOutChan = make(chan *net4go.Conn, options.PlayerBuffer)
	this.closeChan = make(chan struct{})

	game.RunInRoom(this)

	var ticker = time.NewTicker(time.Second / time.Duration(game.Frequency()))

	for {
		select {
		case msg, ok := <-this.messageChan:
			if ok == false {
				return nil
			}
			var player = this.GetPlayer(msg.PlayerId)
			if player != nil {
				game.OnMessage(player, msg.Packet)
			}
		case <-ticker.C:
			if game.OnTick(time.Now().Unix()) == false {
				return nil
			}
		case c, ok := <-this.playerInChan:
			if ok == false {
				return nil
			}
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Online(c)
				game.OnJoinGame(player)
			}
		case c, ok := <-this.playerOutChan:
			if ok == false {
				return nil
			}
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Close()
				game.OnLeaveGame(player)
			}
		case <-this.closeChan:
			return nil
		}
	}
}

// --------------------------------------------------------------------------------
func (this *room) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	var playerId = c.Get(kPlayerId).(uint64)
	if playerId == 0 {
		return false
	}

	var msg = &message{}
	msg.PlayerId = playerId
	msg.Packet = p

	select {
	case this.messageChan <- msg:
	default:
	}

	return true
}

func (this *room) OnClose(c *net4go.Conn, err error) {
	select {
	case this.playerOutChan <- c:
	default:
	}
}

// --------------------------------------------------------------------------------
func (this *room) SendMessage(playerId uint64, p net4go.Packet) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		player.SendMessage(p)
	}
}

func (this *room) Broadcast(p net4go.Packet) {
	this.mu.RLock()
	defer this.mu.RUnlock()

	for _, player := range this.players {
		player.SendMessage(p)
	}
}

// --------------------------------------------------------------------------------
func (this *room) CheckAllPlayerOnline() bool {
	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.players) == 0 {
		return false
	}

	for _, p := range this.players {
		if p.IsOnline() == false {
			return false
		}
	}
	return true
}

func (this *room) CheckAllPlayerReady() bool {
	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.players) == 0 {
		return false
	}

	for _, p := range this.players {
		if p.IsReady() == false {
			return false
		}
	}
	return true
}

func (this *room) GetPlayerCount() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.players)
}

func (this *room) GetOnlinePlayerCount() int {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var i = 0
	for _, p := range this.players {
		if p.IsOnline() {
			i++
		}
	}
	return i
}

func (this *room) GetReadyPlayerCount() int {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var i = 0
	for _, p := range this.players {
		if p.IsReady() {
			i++
		}
	}
	return i
}

func (this *room) Close() error {
	if atomic.LoadUint32(&this.state) == kRoomStateClose {
		return nil
	}
	if atomic.LoadUint32(&this.state) == kRoomStateRunning {
		close(this.messageChan)
		close(this.playerInChan)
		close(this.playerOutChan)
		close(this.closeChan)

		this.messageChan = nil
		this.playerInChan = nil
		this.playerOutChan = nil
	}
	atomic.StoreUint32(&this.state, kRoomStateClose)

	for _, p := range this.players {
		p.Close()
	}
	this.players = make(map[uint64]Player)

	return nil
}
