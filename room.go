package newbee

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"sync"
	"time"
)

const (
	kPlayerId = "player_id"
)

type message struct {
	PlayerId uint64
	Packet   net4go.Packet
}

// --------------------------------------------------------------------------------
type Room struct {
	id      uint64
	mu      sync.RWMutex
	players map[uint64]*Player

	game *Game

	messageChan   chan *message
	playerInChan  chan *net4go.Conn
	playerOutChan chan *net4go.Conn
}

func NewRoom(players []*Player) *Room {
	var r = &Room{}
	r.id = 9999 // TODO 房间 id 生成规则
	r.players = make(map[uint64]*Player)
	for _, player := range players {
		r.players[player.GetId()] = player
	}
	r.messageChan = make(chan *message, 1024)

	r.playerInChan = make(chan *net4go.Conn, 10)
	r.playerOutChan = make(chan *net4go.Conn, 10)

	return r
}

func (this *Room) RunGame(game *Game) {
	if this.game != nil || game == nil {
		return
	}

	this.game = game
	go this.run()
}

// --------------------------------------------------------------------------------
func (this *Room) GetId() uint64 {
	return this.id
}

func (this *Room) GetPlayer(playerId uint64) *Player {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var p = this.players[playerId]
	return p
}

func (this *Room) GetPlayers() []*Player {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var ps = make([]*Player, 0, len(this.players))
	for _, player := range this.players {
		ps = append(ps, player)
	}
	return ps
}

// --------------------------------------------------------------------------------
// Connect 将玩家和连接进行绑定
func (this *Room) Connect(playerId uint64, c *net4go.Conn) {
	if c != nil {
		c.Set(kPlayerId, playerId)
		c.SetHandler(this)
		this.playerInChan <- c
	}
}

// Join 加入新的玩家，如果连接不为空，则将该玩家和连接进行绑定
func (this *Room) Join(player *Player, c *net4go.Conn) {
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
func (this *Room) run() {
	defer func() {
		fmt.Println("游戏停止，房间解散")
		this.game = nil
	}()

	if this.game == nil {
		return
	}

	var ticker = time.NewTicker(time.Second / time.Duration(this.game.Frequency()))

	for {
		select {
		case msg := <-this.messageChan:
			var player = this.GetPlayer(msg.PlayerId)
			if player != nil {
				this.game.ProcessMessage(player, msg.Packet)
			}
		case <-ticker.C:
			if this.game.Tick(time.Now().Unix()) == false {
				return
			}
		case c := <-this.playerInChan:
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Online(c)
			}
		case c := <-this.playerOutChan:
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Close()
			}
		}
	}
}

// --------------------------------------------------------------------------------
func (this *Room) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	var playerId = c.Get(kPlayerId).(uint64)
	if playerId == 0 {
		return false
	}

	var msg = &message{}
	msg.PlayerId = playerId
	msg.Packet = p

	this.messageChan <- msg

	return true
}

func (this *Room) OnClose(c *net4go.Conn, err error) {
	this.playerOutChan <- c
}

// --------------------------------------------------------------------------------
// SendMessage 向指定玩家发送消息
func (this *Room) SendMessage(playerId uint64, p net4go.Packet) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		player.SendMessage(p)
	}
}

// Broadcast 向所有玩家发送消息
func (this *Room) Broadcast(p net4go.Packet) {
	this.mu.RLock()
	defer this.mu.RUnlock()

	for _, player := range this.players {
		player.SendMessage(p)
	}
}

// --------------------------------------------------------------------------------

// CheckAllPlayerOnline 检测所有玩家是否在线
func (this *Room) CheckAllPlayerOnline() bool {
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

// CheckAllReady 检测所有玩家是否准备就绪
func (this *Room) CheckAllPlayerReady() bool {
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

func (this *Room) GetPlayerCount() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.players)
}

// GetOnlinePlayerCount 获取在线玩家数量
func (this *Room) GetOnlinePlayerCount() int {
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

// GetReadyPlayerCount 获取准备就绪玩家数量
func (this *Room) GetReadyPlayerCount() int {
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
