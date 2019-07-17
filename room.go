package newbee

import (
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"sync"
	"time"
)

const (
	kPlayerId = "player_id"
)

const (
	kTimeoutTime = time.Minute * 5 //超时时间
)

type message struct {
	PlayerId uint64
	Packet   *protocol.Packet
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

func newRoom(players []*Player) *Room {
	var r = &Room{}
	r.players = make(map[uint64]*Player)
	for _, player := range players {
		r.players[player.GetId()] = player
	}
	r.game = newGame(r)
	r.messageChan = make(chan *message, 1024)

	r.playerInChan = make(chan *net4go.Conn, 32)
	r.playerOutChan = make(chan *net4go.Conn, 32)
	return r
}

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

func (this *Room) Join(playerId uint64, c *net4go.Conn) {
	c.Set(kPlayerId, playerId)
	c.SetHandler(this)
	this.playerInChan <- c
}

func (this *Room) run() {
	defer func() {

	}()

	var ticker = time.NewTicker(time.Second / time.Duration(this.game.Frequency()))

	for {
		select {
		case msg := <-this.messageChan:
			this.game.ProcessMessage(msg.PlayerId, msg.Packet)
		case <-ticker.C:
			if this.game.Tick(time.Now().Unix()) == false {
				return
			}
		case c := <-this.playerInChan:
			var playerId = c.Get(kPlayerId).(uint64)
			var player = this.GetPlayer(playerId)
			if player != nil {
				player.Connect(c)
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
	switch v := p.(type) {
	case *protocol.Packet:
		var playerId = c.Get(kPlayerId).(uint64)
		if playerId == 0 {
			return false
		}

		var msg = &message{}
		msg.PlayerId = playerId
		msg.Packet = v

		this.messageChan <- msg

		return true
	}
	return false
}

func (this *Room) OnClose(c *net4go.Conn, err error) {
	this.playerOutChan <- c
}

// --------------------------------------------------------------------------------

// --------------------------------------------------------------------------------
func (this *Room) SendMessage(playerId uint64, p net4go.Packet) {
	var player = this.GetPlayer(playerId)
	if player != nil {
		player.SendMessage(p)
	}
}

func (this *Room) Broadcast(p net4go.Packet) {
	for _, player := range this.players {
		player.SendMessage(p)
	}
}

func (this *Room) BroadcastWithoutPlayer(playerId uint64, p net4go.Packet) {
	for _, player := range this.players {
		if player.GetId() == playerId {
			continue
		}
		player.SendMessage(p)
	}
}
