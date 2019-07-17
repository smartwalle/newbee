package newbee

import (
	"fmt"
	"github.com/smartwalle/newbee/protocol"
)

type Game struct {
	room *Room
}

func newGame(room *Room) *Game {
	var g = &Game{}
	g.room = room
	return g
}

// Frequency 帧率，每秒钟发多少数据帧
func (this *Game) Frequency() uint8 {
	return 30
}

func (this *Game) ProcessMessage(playerId uint64, p *protocol.Packet) {
	switch p.GetType() {
	case protocol.PT_HEARTBEAT_REQ:
		this.room.SendMessage(playerId, protocol.NewPacket(protocol.PT_HEARTBEAT_RSP, nil))
	case protocol.PT_LOAD_PROGRESS_REQ:
		var req = &protocol.C2SLoadProgressReq{}
		if err := p.UnmarshalProtoMessage(req); err != nil {
		}
		fmt.Println("加入房间进度", playerId, req.Progress)

		for _, player := range this.room.players {
			player.SendMessage(protocol.NewPacket(protocol.PT_LOAD_PROGRESS_RSP, &protocol.S2CLoadProgressRsp{
				PlayerId: playerId,
				Progress: req.Progress,
			}))
		}
	}
}

func (this *Game) Tick(now int64) bool {
	return true
}
