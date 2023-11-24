package newbee

import (
	"runtime/debug"
	"time"
)

type frameRoom struct {
	*room
	timer *time.Timer
}

func newFrameRoom(room *room) roomMode {
	var r = &frameRoom{}
	r.room = room
	return r
}

func (r *frameRoom) Run(game Game) (err error) {
	//if game == nil {
	//	return ErrNilGame
	//}
	//r.mu.Lock()
	//
	//if r.state == RoomStateClose {
	//	r.mu.Unlock()
	//	return ErrRoomClosed
	//}
	//
	//if r.state == RoomStateRunning {
	//	r.mu.Unlock()
	//	return ErrRoomRunning
	//}
	//
	//r.state = RoomStateRunning
	//r.mu.Unlock()
	//
	//game.OnRunInRoom(r)

	var d = game.TickInterval()
	if d <= 0 {
		return ErrBadInterval
	}
	r.tick(d)

	var mList []*message

	defer func() {
		if r.timer != nil {
			r.timer.Stop()
			r.timer = nil
		}
		game.OnCloseRoom(r)
		r.clean()
	}()

	defer func() {
		if v := recover(); v != nil {
			err = newStackError(v, debug.Stack())

			r.room.panic(game, err)
		}
	}()

RunLoop:
	for {
		select {
		case <-r.timer.C:
			mList = mList[0:0]
			var ok = r.queue.Dequeue(&mList)

			for _, m := range mList {
				//if m == nil {
				//	break RunLoop
				//}

				switch m.Type {
				case mTypeDefault:
					r.onMessage(game, m.PlayerId, m.Data)
				case mTypeCustom:
					r.onDequeue(game, m.Data)
				case mTypePlayerIn:
					m.rError <- r.onJoinRoom(game, m.Player)
				case mTypePlayerOut:
					r.onLeaveRoom(game, m.PlayerId, m.Error)
				}
				r.releaseMessage(m)
			}

			if !ok {
				break RunLoop
			}

			game.OnTick()
			r.tick(d)
		}
	}
	return
}

func (r *frameRoom) tick(d time.Duration) {
	if r.timer == nil {
		r.timer = time.NewTimer(d)
	} else {
		if !r.timer.Stop() {
			select {
			case <-r.timer.C:
			default:
			}
		}
		r.timer.Reset(d)
	}
}

func (r *frameRoom) OnClose() error {
	return nil
}
