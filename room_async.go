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
		mList = mList[0:0]
		this.mQueue.Dequeue(&mList)

		for _, m := range mList {
			if m == nil {
				break RunLoop
			}

			switch m.Type {
			case mTypeDefault:
				this.onMessage(game, m.PlayerId, m.Data)
			case mTypeCustom:
				this.onDequeue(game, m.Data)
			case mTypePlayerIn:
				this.onJoinRoom(game, m.PlayerId)
			case mTypePlayerOut:
				this.onLeaveRoom(game, m.PlayerId)
			}
			this.releaseMessage(m)
		}
	}
	close(stopTicker)

	<-tickerDone

	game.OnCloseRoom(this)
	this.clean()
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

func (this *asyncRoom) OnClose() error {
	return nil
}
