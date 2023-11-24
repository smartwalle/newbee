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
		r.queue = newBlockQueue()
		r.mode = newSyncRoom(r)
	}
}

// WithAsync 网络消息和定时器消息为异步模式
// 网络消息会放入队列中，定时器消息不会放入队列中
// 定时器会定时触发，不管上一次的定时任务是否处理完成
func WithAsync() RoomOption {
	return func(r *room) {
		r.queue = newBlockQueue()
		r.mode = newAsyncRoom(r)
	}
}

// WithFrame 网络消息和定时器消息为帧模式，同时网络消息的处理由定时器驱动
// 会启用一个定时器定时处理网络消息，网络消息处理完成之后，会触发游戏的 OnTick 方法
func WithFrame() RoomOption {
	return func(r *room) {
		r.queue = newQueue()
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
	queue       iMessageQueue
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

	if r.queue == nil {
		r.queue = newBlockQueue()
		r.mode = newAsyncRoom(r)
	}

	return r
}

func (r *room) newMessage(playerId int64, mType messageType, data interface{}, err error) *message {
	if r.messagePool == nil {
		return nil
	}
	var m = r.messagePool.Get().(*message)
	m.Type = mType
	m.PlayerId = playerId
	m.Player = nil
	m.Data = data
	m.Error = err
	m.rError = nil
	return m
}

func (r *room) releaseMessage(m *message) {
	if m != nil && r.messagePool != nil {
		m.Type = 0
		m.PlayerId = 0
		m.Player = nil
		m.Data = nil
		m.Error = nil
		m.rError = nil
		r.messagePool.Put(m)
	}
}

func (r *room) GetId() int64 {
	return r.id
}

func (r *room) GetToken() string {
	return r.token
}

func (r *room) GetState() RoomState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

func (r *room) GetPlayer(playerId int64) Player {
	if playerId == 0 {
		return nil
	}

	r.mu.RLock()
	var p = r.players[playerId]
	r.mu.RUnlock()
	return p
}

func (r *room) popPlayer(playerId int64) Player {
	if playerId == 0 {
		return nil
	}

	r.mu.Lock()
	var p, ok = r.players[playerId]
	if ok {
		delete(r.players, playerId)
	}
	r.mu.Unlock()
	return p
}

func (r *room) GetPlayers() map[int64]Player {
	r.mu.RLock()
	var ps = make(map[int64]Player, len(r.players))
	for pId, p := range r.players {
		ps[pId] = p
	}
	r.mu.RUnlock()
	return ps
}

func (r *room) RangePlayer(fn func(player Player)) {
	r.mu.RLock()
	for _, p := range r.players {
		if p != nil {
			fn(p)
		}
	}
	r.mu.RUnlock()
}

func (r *room) GetPlayerCount() int {
	r.mu.RLock()
	var c = len(r.players)
	r.mu.RUnlock()
	return c
}

func (r *room) AddPlayer(player Player) error {
	if player == nil {
		return ErrNilPlayer
	}

	if player.GetId() == 0 {
		return ErrInvalidPlayer
	}

	if !player.Connected() {
		return ErrBadSession
	}

	r.mu.Lock()
	if r.state != RoomStateRunning {
		r.mu.Unlock()
		return ErrRoomNotRunning
	}

	r.mu.Unlock()

	return r.enqueuePlayerIn(player)

	//// 如果玩家已经存在，则返回错误信息
	//if _, ok := r.players[player.GetId()]; ok {
	//	r.mu.Unlock()
	//	return ErrPlayerExists
	//}
	//
	//r.players[player.GetId()] = player
	//
	//r.mu.Unlock()
	//
	//var sess = player.Session()
	//sess.Set(kPlayerId, player.GetId())
	//r.enqueuePlayerIn(player.GetId())
	//sess.UpdateHandler(r)
	//return nil
}

func (r *room) RemovePlayer(playerId int64) {
	r.enqueuePlayerOut(playerId, nil)
}

func (r *room) Run(game Game) (err error) {
	if game == nil {
		return ErrNilGame
	}
	r.mu.Lock()

	if r.state == RoomStateClose {
		r.mu.Unlock()
		return ErrRoomClosed
	}

	if r.state == RoomStateRunning {
		r.mu.Unlock()
		return ErrRoomRunning
	}

	r.state = RoomStateRunning
	r.closed = make(chan struct{}, 1)
	r.mu.Unlock()

	game.OnRunInRoom(r)

	r.waiter.Add(1)
	defer r.waiter.Done()

	return r.mode.Run(game)
}

func (r *room) OnMessage(sess net4go.Session, p net4go.Packet) {
	var playerId = sess.GetId()
	if playerId == 0 {
		sess.Close()
		return
	}

	var m = r.newMessage(playerId, mTypeDefault, p, nil)
	if m != nil {
		r.queue.Enqueue(m)
	}
}

func (r *room) OnClose(sess net4go.Session, err error) {
	var playerId = sess.GetId()
	if playerId == 0 {
		return
	}

	sess.UpdateHandler(nil)

	r.enqueuePlayerOut(playerId, err)
}

func (r *room) Enqueue(message interface{}) {
	var m = r.newMessage(0, mTypeCustom, message, nil)
	if m != nil {
		r.queue.Enqueue(m)
	}
}

func (r *room) enqueuePlayerIn(player Player) error {
	var m = r.newMessage(player.GetId(), mTypePlayerIn, nil, nil)
	if m != nil {
		var rErr = make(chan error, 1)
		m.Player = player
		m.rError = rErr
		r.queue.Enqueue(m)

		var err error
		select {
		case err = <-rErr:
		case <-r.closed:
			err = ErrRoomClosed
		}
		close(rErr)
		return err
	}
	return nil
}

func (r *room) enqueuePlayerOut(playerId int64, err error) {
	var m = r.newMessage(playerId, mTypePlayerOut, nil, err)
	if m != nil {
		r.queue.Enqueue(m)
	}
}

func (r *room) SendPacket(playerId int64, packet net4go.Packet) {
	var p = r.GetPlayer(playerId)
	if p != nil {
		p.SendPacket(packet)
	}
}

func (r *room) BroadcastPacket(packet net4go.Packet) {
	r.mu.RLock()
	for _, p := range r.players {
		p.SendPacket(packet)
	}
	r.mu.RUnlock()
}

func (r *room) Closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state == RoomStateClose
}

func (r *room) Close() error {
	r.mu.Lock()
	if r.state == RoomStateClose {
		r.mu.Unlock()
		return nil
	}
	r.state = RoomStateClose

	for _, p := range r.players {
		if p != nil {
			r.enqueuePlayerOut(p.GetId(), nil)
		}
	}
	//if r.queue != nil {
	//	r.queue.Enqueue(nil)
	//}
	r.queue.Close()
	r.mu.Unlock()

	var err error
	if r.mode != nil {
		err = r.mode.OnClose()
	}
	return err
}

func (r *room) panic(game Game, err error) {
	game.OnPanic(r, err)

	r.mu.Lock()
	if r.state == RoomStateClose {
		r.mu.Unlock()
		return
	}
	r.state = RoomStateClose

	//var players = r.GetPlayers()
	//for _, p := range players {
	//	if p != nil {
	//		r.onLeaveRoom(game, p.GetId())
	//	}
	//}

	for _, p := range r.players {
		if p == nil {
			continue
		}
		delete(r.players, p.GetId())
		r.mu.Unlock()

		p.Close()
		game.OnLeaveRoom(p, nil)

		r.mu.Lock()
	}
	r.mu.Unlock()

	if r.mode != nil {
		r.mode.OnClose()
	}
}

func (r *room) clean() {
	r.players = nil
	r.messagePool = nil
	r.mode = nil
	close(r.closed)
}
