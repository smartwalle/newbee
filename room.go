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

var (
	ErrRoomClosed     = errors.New("newbee: room is closed")
	ErrRoomRunning    = errors.New("newbee: room is running")
	ErrNilGame        = errors.New("newbee: game is nil")
	ErrPlayerExists   = errors.New("newbee: player already exists")
	ErrPlayerNotExist = errors.New("newbee: player not exist")
	ErrNilPlayer      = errors.New("newbee: player is nil")
	ErrFailedToRun    = errors.New("newbee: failed to run the room")
	ErrBadConnection  = errors.New("newbee: bad connection")
)

type RoomState uint32

const (
	RoomStateClose   RoomState = iota // 房间已关闭
	RoomStatePending                  // 等待游戏运行
	RoomStateRunning                  // 有游戏在运行(和游戏的状态无关，调用 Room 的 Run() 方法成功之后，就会将 Room 的状态调整为此状态)
)

const (
	kDefaultMessageBuffer = 32
	kDefaultPlayerBuffer  = 10
)

type roomOptions struct {
	MessageBuffer int
	PlayerBuffer  int
	Token         string
}

func newRoomOptions() *roomOptions {
	var o = &roomOptions{}
	o.MessageBuffer = kDefaultMessageBuffer
	o.PlayerBuffer = kDefaultPlayerBuffer
	o.Token = ""
	return o
}

type RoomOption func(options *roomOptions)

func WithMessageBuffer(buffer int) RoomOption {
	return func(o *roomOptions) {
		if buffer <= 0 {
			buffer = kDefaultMessageBuffer
		}
		o.MessageBuffer = buffer
	}
}

func WithPlayerBuffer(buffer int) RoomOption {
	return func(o *roomOptions) {
		if buffer <= 0 {
			buffer = kDefaultPlayerBuffer
		}
		o.PlayerBuffer = buffer
	}
}

func WithToken(token string) RoomOption {
	return func(o *roomOptions) {
		o.Token = token
	}
}

type Room interface {
	// GetId 获取房间 id
	GetId() uint64

	// GetToken 房间 token
	GetToken() string

	// GetState 获取房间状态
	GetState() RoomState

	// GetPlayer 获取玩家信息
	GetPlayer(playerId uint64) Player

	// GetPlayers 获取所有玩家信息
	GetPlayers() map[uint64]Player

	// GetPlayersCount 获取玩家数量
	GetPlayersCount() int

	// Connect 将玩家和连接进行绑定
	Connect(playerId uint64, conn net4go.Conn) error

	// Disconnect 断开玩家的网络连接, 作用与 RemovePlayer 一致
	Disconnect(playerId uint64)

	// AddPlayer 加入新的玩家，如果玩家已经存在或者 player 参数为空，会返回相应的错误，如果连接不为空，则将该玩家和连接进行绑定
	AddPlayer(player Player, conn net4go.Conn) error

	// RemovePlayer 移除玩家，如果玩家有网络连接，则会断开网络连接
	RemovePlayer(playerId uint64)

	// Run 启动
	Run(game Game) error

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

	// Close 关闭房间
	Close() error

	// Clean 清理房间
	Clean()
}

type room struct {
	id      uint64
	token   string
	state   RoomState
	mu      sync.RWMutex
	players map[uint64]Player

	//messageChan   chan *message
	//playerInChan  chan *message
	//playerOutChan chan *message
	//closeChan chan struct{}

	closed int32

	mQueue *messageQueue
}

func NewRoom(id uint64, opts ...RoomOption) Room {
	var r = &room{}
	r.id = id
	r.state = RoomStatePending
	r.players = make(map[uint64]Player)

	var options = newRoomOptions()
	for _, opt := range opts {
		opt(options)
	}
	r.token = options.Token
	//r.messageChan = make(chan *message, options.MessageBuffer)
	//r.playerInChan = make(chan *message, options.PlayerBuffer)
	//r.playerOutChan = make(chan *message, options.PlayerBuffer)
	//r.closeChan = make(chan struct{})

	r.closed = 0

	r.mQueue = newQueue()

	return r
}

func (this *room) GetId() uint64 {
	return this.id
}

func (this *room) GetToken() string {
	return this.token
}

func (this *room) GetState() RoomState {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.state
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

func (this *room) GetPlayers() map[uint64]Player {
	this.mu.RLock()
	var ps = make(map[uint64]Player, len(this.players))
	for pId, player := range this.players {
		ps[pId] = player
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) GetPlayersCount() int {
	this.mu.RLock()
	var c = len(this.players)
	this.mu.RUnlock()
	return c
}

func (this *room) Connect(playerId uint64, conn net4go.Conn) error {
	if playerId == 0 {
		return ErrPlayerNotExist
	}

	// 验证玩家是否在本房间
	this.mu.Lock()
	var player = this.players[playerId]
	this.mu.Unlock()

	if player == nil {
		return ErrPlayerNotExist
	}

	if conn == nil || conn.Closed() {
		return ErrBadConnection
	}

	if this.Closed() {
		return ErrRoomClosed
	}

	conn.Set(kPlayerId, playerId)
	conn.UpdateHandler(this)

	var m = newMessage(playerId, messageTypePlayerIn, nil)
	m.Conn = conn

	this.mQueue.Enqueue(m)

	//select {
	//case <-this.closeChan:
	//default:
	//	var m = newMessage(playerId, messageTypePlayerIn, nil)
	//	m.Conn = conn
	//	select {
	//	case this.playerInChan <- m:
	//	case <-time.After(time.Second * 5):
	//	}
	//}
	return nil
}

func (this *room) Disconnect(playerId uint64) {
	//this.mu.Lock()
	//var player = this.players[playerId]
	//this.mu.Unlock()
	//
	//if player != nil {
	//	player.Conn().Close()
	//}
	this.RemovePlayer(playerId)
}

func (this *room) AddPlayer(player Player, conn net4go.Conn) error {
	if player == nil {
		return ErrNilPlayer
	}

	if player.GetId() == 0 {
		return ErrPlayerNotExist
	}

	if this.Closed() {
		return ErrRoomClosed
	}

	this.mu.Lock()

	// 如果玩家已经存在，则返回错误信息
	if _, ok := this.players[player.GetId()]; ok {
		this.mu.Unlock()
		return ErrPlayerExists
	}

	this.players[player.GetId()] = player

	this.mu.Unlock()

	if conn != nil {
		return this.Connect(player.GetId(), conn)
	}
	return nil
}

func (this *room) RemovePlayer(playerId uint64) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		var conn = player.Conn()
		if conn != nil && !conn.Closed() {
			player.Close()
		} else {
			this.mu.Lock()
			delete(this.players, player.GetId())
			this.mu.Unlock()
		}
	}
}

func (this *room) Run(game Game) error {
	if game == nil {
		return ErrNilGame
	}
	this.mu.Lock()

	if this.state == RoomStateClose {
		this.mu.Unlock()
		return ErrRoomClosed
	}

	if this.state == RoomStateRunning {
		this.mu.Unlock()
		return ErrRoomRunning
	}

	this.state = RoomStateRunning
	this.mu.Unlock()

	game.OnRunInRoom(this)

	var stopTicker = make(chan struct{}, 1)
	var tickerDone = make(chan struct{}, 1)

	go this.tick(game, stopTicker, tickerDone)

	var mList []*message
RunLoop:

	for {
		if this.Closed() {
			break RunLoop
		}

		mList = mList[0:0]

		this.mQueue.Dequeue(&mList)

		for _, m := range mList {
			if m == nil || this.Closed() {
				break RunLoop
			}

			var player = this.GetPlayer(m.PlayerId)

			if player == nil {
				continue
			}

			switch m.Type {
			case messageTypeDefault:
				game.OnMessage(player, m.Packet)
			case messageTypePlayerIn:
				player.Connect(m.Conn)
				game.OnJoinRoom(player)
			case messageTypePlayerOut:
				this.mu.Lock()
				delete(this.players, player.GetId())
				this.mu.Unlock()

				game.OnLeaveRoom(player)
				player.Close()
			}
			releaseMessage(m)
		}
	}
	//for {
	//	select {
	//	case <-this.closeChan:
	//		break RunLoop
	//	default:
	//		select {
	//		case <-this.closeChan:
	//			break RunLoop
	//		case m, ok := <-this.messageChan:
	//			if ok == false {
	//				break RunLoop
	//			}
	//			var player = this.GetPlayer(m.PlayerId)
	//			if player != nil {
	//				game.OnMessage(player, m.Packet)
	//			}
	//			releaseMessage(m)
	//		case m, ok := <-this.playerInChan:
	//			if ok == false {
	//				break RunLoop
	//			}
	//			var player = this.GetPlayer(m.PlayerId)
	//			if player != nil {
	//				player.Connect(m.Conn)
	//				game.OnJoinRoom(player)
	//			}
	//		case m, ok := <-this.playerOutChan:
	//			if ok == false {
	//				break RunLoop
	//			}
	//			var player = this.GetPlayer(m.PlayerId)
	//			if player != nil {
	//				this.mu.Lock()
	//				delete(this.players, player.GetId())
	//				this.mu.Unlock()
	//
	//				game.OnLeaveRoom(player)
	//				player.Close()
	//			}
	//			releaseMessage(m)
	//		}
	//	}
	//}
	close(stopTicker)

	<-tickerDone

	game.OnCloseRoom(this)
	this.Close()
	return nil
}

func (this *room) tick(game Game, stopTicker chan struct{}, tickerDone chan struct{}) {
	var t = game.TickInterval()
	if t <= 0 {
		return
	}

	var ticker = time.NewTicker(t)
TickLoop:
	for {
		select {
		case <-stopTicker:
			break TickLoop
		case <-ticker.C:
			if this.Closed() {
				break TickLoop
			}
			if game.OnTick(time.Now().Unix()) == false {
				this.Close()
				break TickLoop
			}
		}
	}
	ticker.Stop()

	close(tickerDone)
}

func (this *room) OnMessage(c net4go.Conn, p net4go.Packet) bool {
	var value = c.Get(kPlayerId)
	if value == nil {
		return false
	}

	var playerId = value.(uint64)
	if playerId == 0 {
		return false
	}

	var m = newMessage(playerId, messageTypeDefault, p)
	this.mQueue.Enqueue(m)

	//select {
	//case <-this.closeChan:
	//	return false
	//default:
	//	var m = newMessage(playerId, messageTypeDefault, p)
	//	select {
	//	case this.messageChan <- m:
	//		return true
	//	case <-time.After(time.Second * 5):
	//		return false
	//	}
	//}
	return true
}

func (this *room) OnClose(c net4go.Conn, err error) {
	var value = c.Get(kPlayerId)
	if value == nil {
		return
	}

	var playerId = value.(uint64)
	if playerId == 0 {
		return
	}

	c.UpdateHandler(nil)

	var m = newMessage(playerId, messageTypePlayerOut, nil)
	this.mQueue.Enqueue(m)

	//select {
	//case <-this.closeChan:
	//default:
	//	var m = newMessage(playerId, messageTypePlayerOut, nil)
	//	select {
	//	case this.playerOutChan <- m:
	//	case <-time.After(time.Second * 5):
	//	}
	//}
}

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

func (this *room) Closed() bool {
	return atomic.LoadInt32(&this.closed) != 0
}

func (this *room) Close() error {
	if old := atomic.SwapInt32(&this.closed, 1); old != 0 {
		return nil
	}

	if this.state == RoomStateClose {
		return nil
	}

	this.state = RoomStateClose

	this.mQueue.Enqueue(nil)

	//close(this.closeChan)
	//
	//close(this.messageChan)
	//close(this.playerInChan)
	//close(this.playerOutChan)
	//
	//this.messageChan = nil
	//this.playerInChan = nil
	//this.playerOutChan = nil

	//for k, p := range this.players {
	//	p.Close()
	//	delete(this.players, k)
	//}

	return nil
}

func (this *room) Clean() {
	this.mu.Lock()
	for k, p := range this.players {
		p.Close()
		delete(this.players, k)
	}
	//if this.mQueue != nil {
	//	this.mQueue.Reset()
	//	this.mQueue = nil
	//}
	this.mu.Unlock()
}
