package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"net"
	"time"
)

func main() {
	c, err := net.Dial("tcp", ":6655")
	if err != nil {
		fmt.Println(err)
		return
	}

	var p = &protocol.Protocol{}
	var h = &ClientHandler{}

	cc := net4go.NewConn(c, p, h)

	var C2SJoinRoom = &protocol.C2SJoinRoom{}
	C2SJoinRoom.RoomId = 9999
	C2SJoinRoom.PlayerId = 1002
	C2SJoinRoom.Token = "token2"

	cc.WritePacket(protocol.NewPacket(protocol.PT_JOIN_ROOM, C2SJoinRoom))

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

func (this *ClientHandler) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
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
						var req = &protocol.C2SLoadProgress{}
						req.Progress = int32(i) * 10
						c.WritePacket(protocol.NewPacket(protocol.PT_LOAD_PROGRESS, req))
						//time.Sleep(time.Second * 1)
					}
				}
			}()
		case protocol.PT_LOAD_PROGRESS:
			var rsp = &protocol.S2CLoadProgress{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}

			fmt.Println("================")
			for _, info := range rsp.Infos {
				fmt.Println("加入房间进度", info.PlayerId, info.Progress)
			}

			var req = &protocol.C2SGameStart{}
			c.WritePacket(protocol.NewPacket(protocol.PT_GAME_START, req))
		case protocol.PT_GAME_START:
			fmt.Println(time.Now().Unix())

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
				for _, data := range frame.FrameData {
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

func (this *ClientHandler) OnClose(c *net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
