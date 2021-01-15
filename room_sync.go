package newbee

import (
	"time"
)

type syncRoom struct {
	*room
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
		if this.Closed() {
			break RunLoop
		}

		mList = mList[0:0]

		this.mQueue.Dequeue(&mList)

		for _, m := range mList {
			if m == nil || this.Closed() {
				break RunLoop
			}

			switch m.Type {
			case messageTypeDefault:
				var player = this.GetPlayer(m.PlayerId)
				if player == nil {
					break
				}
				game.OnMessage(player, m.Packet)
			case messageTypePlayerIn:
				var player = this.GetPlayer(m.PlayerId)
				if player == nil {
					break
				}
				player.Connect(m.Conn)
				game.OnJoinRoom(player)
			case messageTypePlayerOut:
				var player = this.GetPlayer(m.PlayerId)
				if player == nil {
					break
				}
				this.mu.Lock()
				delete(this.players, player.GetId())
				this.mu.Unlock()

				game.OnLeaveRoom(player)
				player.Close()
			case messageTypeTick:
				game.OnTick()
				this.tick(d)
			}
			releaseMessage(m)
		}
	}

	game.OnCloseRoom(this)
	this.Close()
	return nil
}

func (this *syncRoom) tick(d time.Duration) {
	time.AfterFunc(d, func() {
		var m = newMessage(0, messageTypeTick, nil)
		this.mQueue.Enqueue(m)
	})
}
