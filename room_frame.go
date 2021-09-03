package newbee

import (
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

func (this *frameRoom) Run(game Game) error {
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
	if d <= 0 {
		return ErrBadInterval
	}
	this.tick(d)

	var mList []*message

RunLoop:
	for {
		select {
		case <-this.frame:
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
				}
				this.releaseMessage(m)
			}
			game.OnTick()
			this.tick(d)
		}
	}
	if this.timer != nil {
		this.timer.Stop()
	}
	game.OnCloseRoom(this)
	this.clean()
	close(this.frame)
	return nil
}

func (this *frameRoom) tick(d time.Duration) {
	this.timer = time.AfterFunc(d, func() {
		this.frame <- struct{}{}
	})
}

func (this *frameRoom) OnClose() error {
	return nil
}
