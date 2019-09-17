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
	ErrRoomClosed   = errors.New("newbee: room is closed")
	ErrRoomRunning  = errors.New("newbee: room is running")
	ErrNilGame      = errors.New("newbee: game is nil")
	ErrPlayerExists = errors.New("newbee: player already exists")
	ErrNilPlayer    = errors.New("newbee: player is nil")
)

type RoomState uint32

const (
	RoomStateClose   RoomState = iota // 房间已关闭
	RoomStatePending                  // 等待游戏运行
	RoomStateRunning                  // 有游戏在运行(和游戏的状态无关，调用 Room 的 RunGame() 方法成功之后，就会将 Room 的状态调整为此状态)
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

	// GetState 获取房间状态
	GetState() RoomState

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

	// Disconnect 断开玩家的网络连接，但是不会主动将玩家的信息从房间中清除
	Disconnect(playerId uint64)

	// JoinPlayer 加入新的玩家，如果玩家已经存在或者 player 参数为空，会返回相应的错误，如果连接不为空，则将该玩家和连接进行绑定
	JoinPlayer(player Player, conn net4go.Conn) error

	// RemovePlayer 将玩家从房间中移除，只会清除玩家信息，不会断开玩家的网络连接
	RemovePlayer(playerId uint64)

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
	id      uint64
	state   uint32
	mu      sync.RWMutex
	players map[uint64]Player

	messageChan   chan *message
	playerInChan  chan net4go.Conn
	playerOutChan chan *message

	closeChan chan struct{}
}

func NewRoom(roomId uint64, players []Player) Room {
	var r = &room{}
	r.id = roomId
	r.state = uint32(RoomStatePending)
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

func (this *room) GetState() RoomState {
	var s = atomic.LoadUint32(&this.state)
	return RoomState(s)
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

func (this *room) Disconnect(playerId uint64) {
	this.mu.Lock()
	var player = this.players[playerId]
	this.mu.Unlock()

	if player != nil {
		player.Conn().Close()
	}
}

func (this *room) JoinPlayer(player Player, c net4go.Conn) error {
	if player == nil {
		return ErrNilPlayer
	}

	this.mu.Lock()

	// 如果玩家已经存在，则返回错误信息
	if _, ok := this.players[player.GetId()]; ok {
		this.mu.Unlock()
		return ErrPlayerExists
	}

	this.players[player.GetId()] = player

	this.mu.Unlock()

	if c != nil {
		this.Connect(player.GetId(), c)
	}
	return nil
}

func (this *room) RemovePlayer(playerId uint64) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		this.mu.Lock()
		delete(this.players, player.GetId())
		this.mu.Unlock()
	}
}

// --------------------------------------------------------------------------------
func (this *room) RunGame(game Game, opts ...RoomOption) error {
	if game == nil {
		return ErrNilGame
	}

	if RoomState(atomic.LoadUint32(&this.state)) == RoomStateClose {
		return ErrRoomClosed
	}

	if RoomState(atomic.LoadUint32(&this.state)) == RoomStateRunning {
		return ErrRoomRunning
	}

	if atomic.CompareAndSwapUint32(&this.state, uint32(RoomStatePending), uint32(RoomStateRunning)) {
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
					game.OnLeaveGame(player)
					player.Close()
				}
				releaseMessage(m)
			case <-this.closeChan:
				break RunFor
			}
		}
		game.OnCloseRoom()
		this.Close()
	}
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
	var state = RoomState(atomic.LoadUint32(&this.state))

	if state == RoomStateClose {
		return nil
	}

	if atomic.CompareAndSwapUint32(&this.state, uint32(RoomStateRunning), uint32(RoomStateClose)) {
		close(this.messageChan)
		close(this.playerInChan)
		close(this.playerOutChan)
		close(this.closeChan)

		this.messageChan = nil
		this.playerInChan = nil
		this.playerOutChan = nil
	}

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
