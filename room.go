package newbee

import (
	"errors"
	"github.com/smartwalle/net4go"
	"sync"
)

var (
	ErrRoomClosed     = errors.New("newbee: room is closed")
	ErrRoomRunning    = errors.New("newbee: room is running")
	ErrRoomNotRunning = errors.New("newbee: room is not running")
	ErrNilGame        = errors.New("newbee: game is nil")
	ErrPlayerExists   = errors.New("newbee: player already exists")
	ErrPlayerNotExist = errors.New("newbee: player not exist")
	ErrInvalidPlayer  = errors.New("newbee: invalid player")
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

type RoomOption func(r *room)

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

	// AddPlayer 添加玩家，如果玩家已经存在或者 player 参数为空，会返回相应的错误，如果连接不为空，则将该玩家和连接进行绑定
	AddPlayer(player Player) error

	// RemovePlayer 移除玩家
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
	mQueue      iMessageQueue
	waiter      Waiter
	mode        roomMode
	messagePool *sync.Pool
	players     map[int64]Player
	token       string
	id          int64
	mu          sync.RWMutex
	state       RoomState
	closed      chan struct{}
}

func NewRoom(id int64, opts ...RoomOption) Room {
	var r = &room{}
	r.id = id
	r.state = RoomStatePending
	r.players = make(map[int64]Player)
	r.messagePool = &sync.Pool{
		New: func() interface{} {
			return &message{}
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
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

func (this *room) newMessage(playerId int64, mType messageType, data interface{}, err error) *message {
	if this.messagePool == nil {
		return nil
	}
	var m = this.messagePool.Get().(*message)
	m.Type = mType
	m.PlayerId = playerId
	m.Player = nil
	m.Data = data
	m.Error = err
	m.rError = nil
	return m
}

func (this *room) releaseMessage(m *message) {
	if m != nil && this.messagePool != nil {
		m.Type = 0
		m.PlayerId = 0
		m.Player = nil
		m.Data = nil
		m.Error = nil
		m.rError = nil
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
	var p, ok = this.players[playerId]
	if ok {
		delete(this.players, playerId)
	}
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
		return ErrInvalidPlayer
	}

	if !player.Connected() {
		return ErrBadSession
	}

	this.mu.Lock()
	if this.state != RoomStateRunning {
		this.mu.Unlock()
		return ErrRoomNotRunning
	}

	this.mu.Unlock()

	return this.enqueuePlayerIn(player)

	//// 如果玩家已经存在，则返回错误信息
	//if _, ok := this.players[player.GetId()]; ok {
	//	this.mu.Unlock()
	//	return ErrPlayerExists
	//}
	//
	//this.players[player.GetId()] = player
	//
	//this.mu.Unlock()
	//
	//var sess = player.Session()
	//sess.Set(kPlayerId, player.GetId())
	//this.enqueuePlayerIn(player.GetId())
	//sess.UpdateHandler(this)
	//return nil
}

func (this *room) RemovePlayer(playerId int64) {
	this.enqueuePlayerOut(playerId, nil)
}

func (this *room) Run(game Game) (err error) {
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
	this.closed = make(chan struct{}, 1)
	this.mu.Unlock()

	game.OnRunInRoom(this)

	this.waiter.Add(1)
	defer this.waiter.Done()

	return this.mode.Run(game)
}

func (this *room) OnMessage(sess net4go.Session, p net4go.Packet) {
	var playerId = sess.GetId()
	if playerId == 0 {
		sess.Close()
		return
	}

	var m = this.newMessage(playerId, mTypeDefault, p, nil)
	if m != nil {
		this.mQueue.Enqueue(m)
	}
}

func (this *room) OnClose(sess net4go.Session, err error) {
	var playerId = sess.GetId()
	if playerId == 0 {
		return
	}

	sess.UpdateHandler(nil)

	this.enqueuePlayerOut(playerId, err)
}

func (this *room) Enqueue(message interface{}) {
	var m = this.newMessage(0, mTypeCustom, message, nil)
	if m != nil {
		this.mQueue.Enqueue(m)
	}
}

func (this *room) enqueuePlayerIn(player Player) error {
	var m = this.newMessage(player.GetId(), mTypePlayerIn, nil, nil)
	if m != nil {
		var rErr = make(chan error, 1)
		m.Player = player
		m.rError = rErr
		this.mQueue.Enqueue(m)

		var err error
		select {
		case err = <-rErr:
		case <-this.closed:
			err = ErrRoomClosed
		}
		close(rErr)
		return err
	}
	return nil
}

func (this *room) enqueuePlayerOut(playerId int64, err error) {
	var m = this.newMessage(playerId, mTypePlayerOut, nil, err)
	if m != nil {
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
	return this.state == RoomStateClose
}

func (this *room) Close() error {
	this.mu.Lock()
	if this.state == RoomStateClose {
		this.mu.Unlock()
		return nil
	}
	this.state = RoomStateClose

	for _, p := range this.players {
		if p != nil {
			this.enqueuePlayerOut(p.GetId(), nil)
		}
	}
	//if this.mQueue != nil {
	//	this.mQueue.Enqueue(nil)
	//}
	this.mQueue.Close()
	this.mu.Unlock()

	var err error
	if this.mode != nil {
		err = this.mode.OnClose()
	}
	return err
}

func (this *room) panic(game Game, err error) {
	game.OnPanic(this, err)

	this.mu.Lock()
	if this.state == RoomStateClose {
		this.mu.Unlock()
		return
	}
	this.state = RoomStateClose

	//var players = this.GetPlayers()
	//for _, p := range players {
	//	if p != nil {
	//		this.onLeaveRoom(game, p.GetId())
	//	}
	//}

	for _, p := range this.players {
		if p == nil {
			continue
		}
		delete(this.players, p.GetId())
		this.mu.Unlock()

		p.Close()
		game.OnLeaveRoom(p, nil)

		this.mu.Lock()
	}
	this.mu.Unlock()

	if this.mode != nil {
		this.mode.OnClose()
	}
}

func (this *room) clean() {
	this.players = nil
	this.messagePool = nil
	this.mode = nil
	close(this.closed)
}
