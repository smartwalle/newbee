package newbee

import (
	"runtime/debug"
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

func (this *syncRoom) Run(game Game) (err error) {
	//if game == nil {
	//	return ErrNilGame
	//}
	//this.mu.Lock()
	//
	//if this.state == RoomStateClose {
	//	this.mu.Unlock()
	//	return ErrRoomClosed
	//}
	//
	//if this.state == RoomStateRunning {
	//	this.mu.Unlock()
	//	return ErrRoomRunning
	//}
	//
	//this.state = RoomStateRunning
	//this.mu.Unlock()
	//
	//game.OnRunInRoom(this)

	var d = game.TickInterval()
	if d > 0 {
		this.tick(d)
	}

	var mList []*message

	defer func() {
		if this.timer != nil {
			this.timer.Stop()
			this.timer = nil
		}
		game.OnCloseRoom(this)
		this.clean()
	}()

	defer func() {
		if v := recover(); v != nil {
			err = newStackError(v, debug.Stack())

			this.room.panic(game, err)
		}
	}()

RunLoop:
	for {
		mList = mList[0:0]
		var ok = this.mQueue.Dequeue(&mList)

		for _, m := range mList {
			//if m == nil {
			//	break RunLoop
			//}

			switch m.Type {
			case mTypeDefault:
				this.onMessage(game, m.PlayerId, m.Data)
			case mTypeCustom:
				this.onDequeue(game, m.Data)
			case mTypePlayerIn:
				m.rError <- this.onJoinRoom(game, m.Player)
			case mTypePlayerOut:
				this.onLeaveRoom(game, m.PlayerId, m.Error)
			case mTypeTick:
				game.OnTick()
				this.tick(d)
			}
			this.releaseMessage(m)
		}

		if !ok {
			break RunLoop
		}
	}
	return
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
