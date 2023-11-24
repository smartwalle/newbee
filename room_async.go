package newbee

import (
	"runtime/debug"
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

func (r *asyncRoom) Run(game Game) (err error) {
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

	var stopTicker = make(chan struct{}, 1)
	var tickerDone = make(chan struct{}, 1)

	var mList []*message

	defer func() {
		close(stopTicker)
		<-tickerDone
		game.OnCloseRoom(r)
		r.clean()
	}()

	defer func() {
		if v := recover(); v != nil {
			err = newStackError(v, debug.Stack())

			r.room.panic(game, err)
		}
	}()

	go func() {
		defer func() {
			if v := recover(); v != nil {
				err = newStackError(v, debug.Stack())

				r.room.panic(game, err)

				r.queue.Close()
			}
		}()

		r.tick(game, stopTicker, tickerDone)
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
			}
			r.releaseMessage(m)
		}

		if !ok {
			break RunLoop
		}
	}
	return
}

func (r *asyncRoom) tick(game Game, stopTicker chan struct{}, tickerDone chan struct{}) {
	var t = game.TickInterval()
	if t <= 0 {
		return
	}

	var ticker = time.NewTicker(t)

	defer func() {
		ticker.Stop()
		close(tickerDone)
	}()

TickLoop:
	for {
		select {
		case <-stopTicker:
			break TickLoop
		case <-ticker.C:
			if r.Closed() {
				break TickLoop
			}
			game.OnTick()
		}
	}
}

func (r *asyncRoom) OnClose() error {
	return nil
}
