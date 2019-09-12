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

// --------------------------------------------------------------------------------
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

// --------------------------------------------------------------------------------
type Room interface {
	// GetId 获取房间 id
	GetId() uint64

	// GetPlayer 获取玩家信息
	GetPlayer(playerId uint64) Player

	// GetPlayers 获取所有玩家信息
	GetPlayers() []Player

	// GetPlayersWithType 获取指定类型的所有玩家信息
	GetPlayersWithType(pType uint32) []Player

	// GetOnlinePlayers 获取在线的玩家信息
	GetOnlinePlayers() []Player

	// GetOnlinePlayersWithType 获取在线的玩家信息(指定玩家类型)
	GetOnlinePlayersWithType(pType uint32) []Player

	// GetReadyPlayers 获取准备就绪的玩家信息
	GetReadyPlayers() []Player

	// GetReadyPlayersWithType 获取准备就绪的玩家信息(指定玩家类型)
	GetReadyPlayersWithType(pType uint32) []Player

	// Connect 将玩家和连接进行绑定
	Connect(playerId uint64, conn net4go.Conn)

	// JoinPlayer 加入新的玩家，如果连接不为空，则将该玩家和连接进行绑定
	JoinPlayer(player Player, conn net4go.Conn)

	// RunGame 启动游戏
	RunGame(game Game, opts ...RoomOption) error

	// SendMessage 向指定玩家发送消息
	SendMessage(playerId uint64, b []byte)

	// SendPacket 向指定玩家发送消息
	SendPacket(playerId uint64, packet net4go.Packet)

	// BroadcastMessage 向所有玩家广播消息
	BroadcastMessage(b []byte)

	// BroadcastMessageWithType 向指定类型的玩家广播消息
	BroadcastMessageWithType(pType uint32, b []byte)

	// BroadcastPacket 向所有玩家广播消息
	BroadcastPacket(packet net4go.Packet)

	// BroadcastPacketWithType 向指定类型的玩家广播消息
	BroadcastPacketWithType(pType uint32, p net4go.Packet)

	// CheckPlayerOnline 检测指定玩家是否在线
	CheckPlayerOnline(playerId uint64) bool

	// CheckPlayersOnline 检测所有的玩家是否在线
	CheckPlayersOnline() bool

	// CheckPlayersOnlineWithType 检测所有的玩家是否在线(指定玩家类型)
	CheckPlayersOnlineWithType(pType uint32) bool

	// CheckPlayerReady 检测指定玩家是否准备就绪
	CheckPlayerReady(playerId uint64) bool

	// CheckPlayersReady 检测所有的玩家是否准备就绪
	CheckPlayersReady() bool

	// CheckPlayersReadyWithType 检测所有的玩家是否准备就绪(指定玩家类型)
	CheckPlayersReadyWithType(pType uint32) bool

	// GetPlayersCount 获取玩家数量
	GetPlayersCount() int

	// GetPlayersCountWithType 获取玩家数量(指定玩家类型)
	GetPlayersCountWithType(pType uint32) int

	// GetOnlinePlayersCount 获取在线的玩家数量
	GetOnlinePlayersCount() int

	// GetOnlinePlayersCountWithType 获取在线的玩家数量(指定玩家类型)
	GetOnlinePlayersCountWithType(pType uint32) int

	// GetReadyPlayersCount 获取准备就绪的玩家数量
	GetReadyPlayersCount() int

	// GetReadyPlayersCountWithType 获取准备就绪的玩家数量(指定玩家类型)
	GetReadyPlayersCountWithType(pType uint32) int

	// Close 关闭房间
	Close() error
}

// --------------------------------------------------------------------------------
type room struct {
	id       uint64
	state    uint32
	mu       sync.RWMutex
	players  map[uint64]Player
	watchers map[uint64]Player

	messageChan   chan *message
	playerInChan  chan net4go.Conn
	playerOutChan chan *message

	closeChan chan struct{}
}

func NewRoom(roomId uint64, players []Player) Room {
	var r = &room{}
	r.id = roomId
	r.state = kRoomStatePending
	r.players = make(map[uint64]Player)
	r.watchers = make(map[uint64]Player)
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
	if playerId == 0 {
		return nil
	}

	this.mu.RLock()
	var p = this.players[playerId]
	this.mu.RUnlock()
	return p
}

func (this *room) GetPlayers() []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, player := range this.players {
		ps = append(ps, player)
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetPlayersWithType(pType uint32) []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, player := range this.players {
		if player.GetType() == pType {
			ps = append(ps, player)
		}
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetOnlinePlayers() []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, p := range this.players {
		if p.IsOnline() {
			ps = append(ps, p)
		}
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetOnlinePlayersWithType(pType uint32) []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, p := range this.players {
		if p.GetType() == pType && p.IsOnline() {
			ps = append(ps, p)
		}
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetReadyPlayers() []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, p := range this.players {
		if p.IsReady() {
			ps = append(ps, p)
		}
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetReadyPlayersWithType(pType uint32) []Player {
	this.mu.RLock()
	var ps = make([]Player, 0, len(this.players))
	for _, p := range this.players {
		if p.GetType() == pType && p.IsReady() {
			ps = append(ps, p)
		}
	}
	this.mu.RUnlock()
	return ps
}

// --------------------------------------------------------------------------------
func (this *room) Connect(playerId uint64, c net4go.Conn) {
	if c != nil {
		c.Set(kPlayerId, playerId)
		c.UpdateHandler(this)

		select {
		case this.playerInChan <- c:
		default:
		}
	}
}

func (this *room) JoinPlayer(player Player, c net4go.Conn) {
	if player != nil {
		this.mu.Lock()
		// 玩家不存在则添加该玩家
		if _, ok := this.players[player.GetId()]; ok == false {
			this.players[player.GetId()] = player
		}
		this.mu.Unlock()

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

	var options = newRoomOptions()
	for _, o := range opts {
		o.Apply(options)
	}
	this.messageChan = make(chan *message, options.MessageBuffer)
	this.playerInChan = make(chan net4go.Conn, options.PlayerBuffer)
	this.playerOutChan = make(chan *message, options.PlayerBuffer)
	this.closeChan = make(chan struct{})

	game.RunInRoom(this)

	var ticker = time.NewTicker(time.Second / time.Duration(game.Frequency()))

RunFor:
	for {
		select {
		case m, ok := <-this.messageChan:
			if ok == false {
				break RunFor
			}
			var player = this.GetPlayer(m.PlayerId)
			if player != nil {
				game.OnMessage(player, m.Packet)
			}
			releaseMessage(m)
		case <-ticker.C:
			if game.OnTick(time.Now().Unix()) == false {
				break RunFor
			}
		case c, ok := <-this.playerInChan:
			if ok == false {
				break RunFor
			}
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Online(c)
				game.OnJoinGame(player)
			}
		case m, ok := <-this.playerOutChan:
			if ok == false {
				break RunFor
			}
			var player = this.GetPlayer(m.PlayerId)
			if player != nil {
				player.Close()
				game.OnLeaveGame(player)
			}
			releaseMessage(m)
		case <-this.closeChan:
			break RunFor
		}
	}
	game.OnCloseRoom()
	this.Close()
	return nil
}

// --------------------------------------------------------------------------------
func (this *room) OnMessage(c net4go.Conn, p net4go.Packet) bool {
	var playerId = c.Get(kPlayerId).(uint64)
	if playerId == 0 {
		return false
	}

	var m = newMessage(playerId, p)
	select {
	case this.messageChan <- m:
	default:
	}

	return true
}

func (this *room) OnClose(c net4go.Conn, err error) {
	var playerId = c.Get(kPlayerId).(uint64)
	if playerId == 0 {
		return
	}

	var m = newMessage(playerId, nil)
	select {
	case this.playerOutChan <- m:
	default:
	}
}

// --------------------------------------------------------------------------------
func (this *room) SendMessage(playerId uint64, b []byte) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		player.SendMessage(b)
	}
}

func (this *room) SendPacket(playerId uint64, p net4go.Packet) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		player.SendPacket(p)
	}
}

func (this *room) BroadcastMessage(b []byte) {
	this.mu.RLock()
	for _, player := range this.players {
		player.SendMessage(b)
	}
	this.mu.RUnlock()
}

func (this *room) BroadcastMessageWithType(pType uint32, b []byte) {
	this.mu.RLock()
	for _, player := range this.players {
		if player.GetType() == pType {
			player.SendMessage(b)
		}
	}
	this.mu.RUnlock()
}

func (this *room) BroadcastPacket(p net4go.Packet) {
	this.mu.RLock()
	for _, player := range this.players {
		player.SendPacket(p)
	}
	this.mu.RUnlock()
}

func (this *room) BroadcastPacketWithType(pType uint32, p net4go.Packet) {
	this.mu.RLock()
	for _, player := range this.players {
		if player.GetType() == pType {
			player.SendPacket(p)
		}
	}
	this.mu.RUnlock()
}

// --------------------------------------------------------------------------------
func (this *room) CheckPlayerOnline(playerId uint64) bool {
	this.mu.RLock()
	player, ok := this.players[playerId]
	this.mu.RUnlock()

	if ok == false {
		return false
	}
	return player.IsOnline()
}

func (this *room) CheckPlayersOnline() bool {
	this.mu.RLock()

	if len(this.players) == 0 {
		this.mu.RUnlock()
		return false
	}

	for _, p := range this.players {
		if p.IsOnline() == false {
			this.mu.RUnlock()
			return false
		}
	}
	this.mu.RUnlock()
	return true
}

func (this *room) CheckPlayersOnlineWithType(pType uint32) bool {
	this.mu.RLock()

	if len(this.players) == 0 {
		this.mu.RUnlock()
		return false
	}

	for _, p := range this.players {
		if p.GetType() == pType && p.IsOnline() == false {
			this.mu.RUnlock()
			return false
		}
	}
	this.mu.RUnlock()
	return true
}

func (this *room) CheckPlayerReady(playerId uint64) bool {
	this.mu.RLock()
	player, ok := this.players[playerId]
	this.mu.RUnlock()

	if ok == false {
		return false
	}
	return player.IsReady()
}

func (this *room) CheckPlayersReady() bool {
	this.mu.RLock()

	if len(this.players) == 0 {
		this.mu.RUnlock()
		return false
	}

	for _, p := range this.players {
		if p.IsReady() == false {
			this.mu.RUnlock()
			return false
		}
	}
	this.mu.RUnlock()
	return true
}

func (this *room) CheckPlayersReadyWithType(pType uint32) bool {
	this.mu.RLock()

	if len(this.players) == 0 {
		this.mu.RUnlock()
		return false
	}

	for _, p := range this.players {
		if p.GetType() == pType && p.IsReady() == false {
			this.mu.RUnlock()
			return false
		}
	}
	this.mu.RUnlock()
	return true
}

func (this *room) GetPlayersCount() int {
	this.mu.RLock()
	var c = len(this.players)
	this.mu.RUnlock()
	return c
}

func (this *room) GetPlayersCountWithType(pType uint32) int {
	this.mu.RLock()
	var i = 0
	for _, p := range this.players {
		if p.GetType() == pType {
			i++
		}
	}
	this.mu.RUnlock()
	return i
}

func (this *room) GetOnlinePlayersCount() int {
	this.mu.RLock()
	var i = 0
	for _, p := range this.players {
		if p.IsOnline() {
			i++
		}
	}
	this.mu.RUnlock()
	return i
}

func (this *room) GetOnlinePlayersCountWithType(pType uint32) int {
	this.mu.RLock()
	var i = 0
	for _, p := range this.players {
		if p.GetType() == pType && p.IsOnline() {
			i++
		}
	}
	this.mu.RUnlock()
	return i
}

func (this *room) GetReadyPlayersCount() int {
	this.mu.RLock()
	var i = 0
	for _, p := range this.players {
		if p.IsReady() {
			i++
		}
	}
	this.mu.RUnlock()
	return i
}

func (this *room) GetReadyPlayersCountWithType(pType uint32) int {
	this.mu.RLock()
	var i = 0
	for _, p := range this.players {
		if p.GetType() == pType && p.IsReady() {
			i++
		}
	}
	this.mu.RUnlock()
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

// --------------------------------------------------------------------------------
var messagePool = &sync.Pool{
	New: func() interface{} {
		return &message{}
	},
}

func newMessage(playerId uint64, packet net4go.Packet) *message {
	var m = messagePool.Get().(*message)
	m.PlayerId = playerId
	m.Packet = packet
	return m
}

func releaseMessage(m *message) {
	if m != nil {
		m.PlayerId = 0
		m.Packet = nil
		messagePool.Put(m)
	}
}
