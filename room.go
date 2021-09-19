package newbee

import (
	"errors"
	"github.com/smartwalle/net4go"
	"sync"
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
	ErrBadSession     = errors.New("newbee: bad session")
	ErrBadInterval    = errors.New("newbee: bad interval")
)

type RoomState uint32

const (
	RoomStateClose   RoomState = iota // 房间已关闭
	RoomStatePending                  // 等待游戏运行
	RoomStateRunning                  // 有游戏在运行(和游戏的状态无关，调用 Room 的 Run() 方法成功之后，就会将 Room 的状态调整为此状态)
)

//type roomOptions struct {
//	token string
//	mode  func(*room) roomMode
//}
//
//func newRoomOptions() *roomOptions {
//	var o = &roomOptions{}
//	o.token = ""
//	return o
//}

type RoomOption func(options *room)

func WithToken(token string) RoomOption {
	return func(r *room) {
		r.token = token
	}
}

func WithWaiter(w Waiter) RoomOption {
	return func(r *room) {
		r.waiter = w
	}
}

// WithSync 网络消息和定时器消息为同步模式
// 网络消息和定时器消息会放入同一队列等待执行
// 定时任务放入队列之后，定时器就会暂停，需要等到队列中的定时任务执行之后才会再次激活定时器
func WithSync() RoomOption {
	return func(r *room) {
		r.mQueue = newBlockQueue()
		r.mode = newSyncRoom(r)
	}
}

// WithAsync 网络消息和定时器消息为异步模式
// 网络消息会放入队列中，定时器消息不会放入队列中
// 定时器会定时触发，不管上一次的定时任务是否处理完成
func WithAsync() RoomOption {
	return func(r *room) {
		r.mQueue = newBlockQueue()
		r.mode = newAsyncRoom(r)
	}
}

// WithFrame 网络消息和定时器消息为帧模式，同时网络消息的处理由定时器驱动
// 会启用一个定时器定时处理网络消息，网络消息处理完成之后，会触发游戏的 OnTick 方法
func WithFrame() RoomOption {
	return func(r *room) {
		r.mQueue = newQueue()
		r.mode = newFrameRoom(r)
	}
}

type Room interface {
	// GetId 获取房间 id
	GetId() int64

	// GetToken 房间 token
	GetToken() string

	// GetState 获取房间状态
	GetState() RoomState

	// GetPlayer 获取玩家信息
	GetPlayer(playerId int64) Player

	// GetPlayers 获取所有玩家信息
	GetPlayers() map[int64]Player

	// RangePlayer 只读遍历玩家信息，在回调函数中，不可执行 Room 的其它可以影响玩家列表的操作
	RangePlayer(fn func(player Player))

	// GetPlayerCount 获取玩家数量
	GetPlayerCount() int

	// AddPlayer 加入新的玩家，如果玩家已经存在或者 player 参数为空，会返回相应的错误，如果连接不为空，则将该玩家和连接进行绑定
	AddPlayer(player Player) error

	// RemovePlayer 移除玩家，如果玩家有网络连接，则会断开网络连接
	RemovePlayer(playerId int64)

	// Run 启动
	Run(game Game) error

	// Enqueue 添加自定义消息
	Enqueue(message interface{})

	// SendPacket 向指定玩家发送消息
	SendPacket(playerId int64, packet net4go.Packet)

	// BroadcastPacket 向所有玩家广播消息
	BroadcastPacket(packet net4go.Packet)

	// Close 关闭房间
	Close() error
}

type roomMode interface {
	Run(game Game) error

	OnClose() error
}

type room struct {
	id          int64
	token       string
	waiter      Waiter
	state       RoomState
	mu          sync.RWMutex
	players     map[int64]Player
	closed      bool
	messagePool *sync.Pool

	mQueue iMessageQueue
	mode   roomMode
}

func NewRoom(id int64, opts ...RoomOption) Room {
	var r = &room{}
	r.id = id
	r.state = RoomStatePending
	r.players = make(map[int64]Player)
	r.closed = false
	r.messagePool = &sync.Pool{
		New: func() interface{} {
			return &message{}
		},
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.waiter == nil {
		r.waiter = &sync.WaitGroup{}
	}

	if r.mQueue == nil {
		r.mQueue = newBlockQueue()
		r.mode = newAsyncRoom(r)
	}

	return r
}

func (this *room) newMessage(playerId int64, mType messageType, data interface{}) *message {
	if this.messagePool == nil {
		return nil
	}
	var m = this.messagePool.Get().(*message)
	m.PlayerId = playerId
	m.Type = mType
	m.Data = data
	return m
}

func (this *room) releaseMessage(m *message) {
	if m != nil && this.messagePool != nil {
		m.PlayerId = 0
		m.Type = 0
		m.Data = nil
		this.messagePool.Put(m)
	}
}

func (this *room) GetId() int64 {
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

func (this *room) GetPlayer(playerId int64) Player {
	if playerId == 0 {
		return nil
	}

	this.mu.RLock()
	var p = this.players[playerId]
	this.mu.RUnlock()
	return p
}

func (this *room) popPlayer(playerId int64) Player {
	if playerId == 0 {
		return nil
	}

	this.mu.Lock()
	var p = this.players[playerId]
	delete(this.players, playerId)
	this.mu.Unlock()
	return p
}

func (this *room) GetPlayers() map[int64]Player {
	this.mu.RLock()
	var ps = make(map[int64]Player, len(this.players))
	for pId, p := range this.players {
		ps[pId] = p
	}
	this.mu.RUnlock()
	return ps
}

func (this *room) RangePlayer(fn func(player Player)) {
	this.mu.RLock()
	for _, p := range this.players {
		if p != nil {
			fn(p)
		}
	}
	this.mu.RUnlock()
}

func (this *room) GetPlayerCount() int {
	this.mu.RLock()
	var c = len(this.players)
	this.mu.RUnlock()
	return c
}

func (this *room) AddPlayer(player Player) error {
	if player == nil {
		return ErrNilPlayer
	}

	if player.GetId() == 0 {
		return ErrPlayerNotExist
	}

	if player.Connected() == false {
		return ErrBadSession
	}

	this.mu.Lock()
	if this.closed {
		this.mu.Unlock()
		return ErrRoomClosed
	}

	// 如果玩家已经存在，则返回错误信息
	if _, ok := this.players[player.GetId()]; ok {
		this.mu.Unlock()
		return ErrPlayerExists
	}

	this.players[player.GetId()] = player

	this.mu.Unlock()

	var sess = player.Session()
	sess.Set(kPlayerId, player.GetId())
	this.enqueuePlayerIn(player.GetId())
	sess.UpdateHandler(this)
	return nil
}

func (this *room) RemovePlayer(playerId int64) {
	this.enqueuePlayerOut(playerId)
}

func (this *room) Run(game Game) error {
	defer this.waiter.Done()
	this.waiter.Add(1)
	return this.mode.Run(game)
}

func (this *room) OnMessage(sess net4go.Session, p net4go.Packet) {
	var playerId, _ = sess.Get(kPlayerId).(int64)
	if playerId == 0 {
		sess.Close()
		return
	}

	var m = this.newMessage(playerId, mTypeDefault, p)
	this.mQueue.Enqueue(m)
}

func (this *room) OnClose(sess net4go.Session, err error) {
	var playerId, _ = sess.Get(kPlayerId).(int64)
	if playerId == 0 {
		return
	}

	sess.UpdateHandler(nil)

	this.enqueuePlayerOut(playerId)
}

func (this *room) Enqueue(message interface{}) {
	if this.state != RoomStateClose {
		var m = this.newMessage(0, mTypeCustom, message)
		this.mQueue.Enqueue(m)
	}
}

func (this *room) enqueuePlayerIn(playerId int64) {
	if this.state != RoomStateClose {
		var m = this.newMessage(playerId, mTypePlayerIn, nil)
		this.mQueue.Enqueue(m)
	}
}

func (this *room) enqueuePlayerOut(playerId int64) {
	if this.state != RoomStateClose {
		var m = this.newMessage(playerId, mTypePlayerOut, nil)
		this.mQueue.Enqueue(m)
	}
}

func (this *room) SendPacket(playerId int64, packet net4go.Packet) {
	var p = this.GetPlayer(playerId)
	if p != nil {
		p.SendPacket(packet)
	}
}

func (this *room) BroadcastPacket(packet net4go.Packet) {
	this.mu.RLock()
	for _, p := range this.players {
		p.SendPacket(packet)
	}
	this.mu.RUnlock()
}

func (this *room) Closed() bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.closed
}

func (this *room) Close() error {
	this.mu.Lock()
	if this.closed {
		this.mu.Unlock()
		return nil
	}
	this.closed = true
	this.mu.Unlock()

	if this.state == RoomStateClose {
		return nil
	}

	this.state = RoomStateClose

	this.mu.RLock()
	for _, p := range this.players {
		this.enqueuePlayerOut(p.GetId())
	}
	this.mu.RUnlock()

	this.mQueue.Enqueue(nil)

	return this.mode.OnClose()
}

func (this *room) clean() {
	this.players = nil
	this.messagePool = nil
	this.mode = nil
	this.mQueue = nil
}
