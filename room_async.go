package newbee

import (
	"time"
)

type asyncRoom struct {
	*room
}

func newAsyncRoom(room *room) roomMode {
	var r = &asyncRoom{}
	r.room = room
	return r
}

func (this *asyncRoom) Run(game Game) error {
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
				releaseMessage(m)
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
	close(stopTicker)

	<-tickerDone

	game.OnCloseRoom(this)
	this.Close()
	return nil
}

func (this *asyncRoom) tick(game Game, stopTicker chan struct{}, tickerDone chan struct{}) {
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
			game.OnTick()
		}
	}
	ticker.Stop()

	close(tickerDone)
}
