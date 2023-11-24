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

func (r *syncRoom) Run(game Game) (err error) {
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
	if d > 0 {
		r.tick(d)
	}

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
			case mTypeTick:
				game.OnTick()
				r.tick(d)
			}
			r.releaseMessage(m)
		}

		if !ok {
			break RunLoop
		}
	}
	return
}

func (r *syncRoom) tick(d time.Duration) {
	r.timer = time.AfterFunc(d, func() {
		var m = r.newMessage(0, mTypeTick, nil, nil)
		r.queue.Enqueue(m)
	})
}

func (r *syncRoom) OnClose() error {
	return nil
}
