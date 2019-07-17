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

	var joinRoomReq = &protocol.C2SJoinRoomReq{}
	joinRoomReq.RoomId = 5577006791947779410
	joinRoomReq.PlayerId = 1001
	joinRoomReq.Token = "token1"

	cc.WritePacket(protocol.NewPacket(protocol.PT_JOIN_ROOM_REQ, joinRoomReq))

	go func() {
		for {
			cc.WritePacket(protocol.NewPacket(protocol.PT_HEARTBEAT_REQ, nil))
			time.Sleep(time.Second * 1)
		}
	}()

	select {}
}

type ClientHandler struct {
}

func (this *ClientHandler) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	fmt.Println("OnMessage", p)

	switch v := p.(type) {
	case *protocol.Packet:
		switch v.GetType() {
		case protocol.PT_JOIN_ROOM_RSP:
			var rsp = &protocol.S2CJoinRoomRsp{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}
			fmt.Println("加入房间返回结果", rsp.Code)

			go func() {
				if rsp.Code == protocol.JOIN_ROOM_CODE_SUCCESS {
					for i := 1; i <= 10; i++ {
						var req = &protocol.C2SLoadProgressReq{}
						req.Progress = int32(i) * 10
						c.WritePacket(protocol.NewPacket(protocol.PT_LOAD_PROGRESS_REQ, req))
						time.Sleep(time.Second * 2)
					}
				}
			}()
		case protocol.PT_LOAD_PROGRESS_RSP:
			var rsp = &protocol.S2CLoadProgressRsp{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}
			fmt.Println("加入房间进度", rsp.PlayerId, rsp.Progress)
		case protocol.PT_HEARTBEAT_RSP:
			fmt.Println("收到心跳请求回应")
		}
	}
	return true
}

func (this *ClientHandler) OnClose(c *net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
