package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/sample/protocol"
	"net"
	"time"
)

func main() {
	c, err := net.Dial("tcp", "192.168.1.99:6666")
	if err != nil {
		fmt.Println(err)
		return
	}

	var p = &protocol.Protocol{}
	var h = &ClientHandler{}

	cc := net4go.NewConn(c, p, h)

	var req = &protocol.C2SJoinRoom{}
	req.RoomId = 9999
	req.PlayerId = 1001
	req.Token = "token1"

	cc.WritePacket(protocol.NewPacket(protocol.PT_JOIN_ROOM, req))

	go func() {
		for {
			cc.WritePacket(protocol.NewPacket(protocol.PT_HEARTBEAT, nil))
			time.Sleep(time.Second * 1)
		}
	}()

	select {}
}

type ClientHandler struct {
}

func (this *ClientHandler) OnMessage(c net4go.Conn, p net4go.Packet) bool {
	//fmt.Println("OnMessage", p)

	switch v := p.(type) {
	case *protocol.Packet:
		switch v.GetType() {
		case protocol.PT_JOIN_ROOM:
			var rsp = &protocol.S2CJoinRoom{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}
			fmt.Println("加入房间返回结果", rsp.Code)

			go func() {
				if rsp.Code == protocol.JOIN_ROOM_CODE_SUCCESS {
					for i := 1; i <= 10; i++ {
						var req = &protocol.C2SLoadingProgress{}
						req.Progress = int32(i) * 10
						c.WritePacket(protocol.NewPacket(protocol.PT_LOADING_PROGRESS, req))
						//time.Sleep(time.Second * 1)
					}

					var req = &protocol.C2SGameReady{}
					c.WritePacket(protocol.NewPacket(protocol.PT_GAME_READY, req))
				}
			}()
		case protocol.PT_LOADING_PROGRESS:
			var rsp = &protocol.S2CLoadingProgress{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}

			fmt.Println("================")
			for _, info := range rsp.Infos {
				fmt.Println("加入房间进度", info.PlayerId, info.Progress)
			}
		case protocol.PT_GAME_READY:
			var req = &protocol.C2SGameFrame{}
			req.FrameId = 0
			req.PlayerMove = &protocol.PlayerMove{X: 10, Y: 11}
			c.WritePacket(protocol.NewPacket(protocol.PT_GAME_FRAME, req))

		case protocol.PT_GAME_FRAME:
			var rsp = &protocol.S2CGameFrame{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}

			fmt.Println("============ frame")
			for _, frame := range rsp.Frames {
				fmt.Println("frame id", frame.FrameId)
				for _, data := range frame.Commands {
					fmt.Println(data.PlayerId, data.PlayerMove, data.PlayerSkill)
				}
			}

			var req = &protocol.C2SGameFrame{}
			req.FrameId = rsp.Frames[len(rsp.Frames)-1].FrameId + 1
			req.PlayerMove = &protocol.PlayerMove{X: 10, Y: 11}
			c.WritePacket(protocol.NewPacket(protocol.PT_GAME_FRAME, req))

		case protocol.PT_HEARTBEAT:
		}
	}
	return true
}

func (this *ClientHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
