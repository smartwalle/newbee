package newbee

import (
	"runtime/debug"
	"time"
)

type frameRoom struct {
	*room
	frame chan struct{}
	timer *time.Timer
}

func newFrameRoom(room *room) roomMode {
	var r = &frameRoom{}
	r.room = room
	r.frame = make(chan struct{}, 1)
	return r
}

func (this *frameRoom) Run(game Game) (err error) {
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
	if d <= 0 {
		return ErrBadInterval
	}
	this.tick(d)

	var mList []*message

	defer func() {
		if this.timer != nil {
			this.timer.Stop()
			this.timer = nil
		}
		game.OnCloseRoom(this)
		this.clean()
		close(this.frame)
	}()

	defer func() {
		if v := recover(); v != nil {
			err = newStackError(v, debug.Stack())

			this.room.panic(game, err)
		}
	}()

RunLoop:
	for {
		select {
		case <-this.frame:
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
				}
				this.releaseMessage(m)
			}

			if ok == false {
				break RunLoop
			}

			game.OnTick()
			this.tick(d)
		}
	}
	return
}

func (this *frameRoom) tick(d time.Duration) {
	this.timer = time.AfterFunc(d, func() {
		this.frame <- struct{}{}
	})
}

func (this *frameRoom) OnClose() error {
	return nil
}
