package newbee

import (
	"time"
)

type syncRoom struct {
	*room
	timer *time.Timer
}

func newSyncRoom(room *room) roomMode {
	var r = &syncRoom{}
	r.room = room
	return r
}

func (this *syncRoom) Run(game Game) error {
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

	var d = game.TickInterval()
	if d > 0 {
		this.tick(d)
	}

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
				if this.onMessage(game, m.PlayerId, m.Data) {
					break
				}
			case mTypeCustom:
				if this.onDequeue(game, m.Data) {
					break
				}
			case mTypePlayerIn:
				if this.onJoinRoom(game, m.PlayerId, m.Session) {
					break
				}
			case mTypePlayerOut:
				if this.onLeaveRoom(game, m.PlayerId) {
					break
				}
			case mTypeTick:
				game.OnTick()
				this.tick(d)
			}
			this.releaseMessage(m)
		}
	}
	if this.timer != nil {
		this.timer.Stop()
	}
	game.OnCloseRoom(this)
	this.clean()
	return nil
}

func (this *syncRoom) tick(d time.Duration) {
	this.timer = time.AfterFunc(d, func() {
		var m = this.newMessage(0, mTypeTick, nil, nil)
		this.mQueue.Enqueue(m)
	})
}

func (this *syncRoom) OnClose() error {
	return nil
}
