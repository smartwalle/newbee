package newbee

import (
	"time"
)

type frameRoom struct {
	*room
	frame chan struct{}
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
		if this.Closed() {
			break RunLoop
		}
		select {
		case _, ok := <-this.frame:
			if ok == false {
				break RunLoop
			}

			mList = mList[0:0]

			this.mQueue.Dequeue(&mList)

			for _, m := range mList {
				if m == nil || this.Closed() {
					break RunLoop
				}

				switch m.Type {
				case mTypeDefault:
					var player = this.GetPlayer(m.PlayerId)
					if player == nil {
						break
					}
					game.OnMessage(player, m.Packet)
				case mTypePlayerIn:
					var player = this.GetPlayer(m.PlayerId)
					if player == nil {
						break
					}
					player.Connect(m.Conn)
					game.OnJoinRoom(player)
				case mTypePlayerOut:
					var player = this.GetPlayer(m.PlayerId)
					if player == nil {
						break
					}
					this.mu.Lock()
					delete(this.players, player.GetId())
					this.mu.Unlock()

					game.OnLeaveRoom(player)
					player.Close()
					//case mTypeTick:
					//	game.OnTick()
					//	this.tick(d)
				}
				releaseMessage(m)
			}
			game.OnTick()
			this.tick(d)
		}
	}

	game.OnCloseRoom(this)
	this.Close()
	return nil
}

func (this *frameRoom) tick(d time.Duration) {
	time.AfterFunc(d, func() {
		this.frame <- struct{}{}
	})
}

func (this *frameRoom) OnClose() error {
	close(this.frame)
	return nil
}
