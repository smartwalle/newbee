package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"net"
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


	var pJoinRoom = &protocol.C2SJoinRoomReq{}
	pJoinRoom.PlayerId = 1009
	pJoinRoom.Token = "这是我的 Token"

	cc.WritePacket(protocol.NewPacket(1000, pJoinRoom))

	select {
	}
}

type ClientHandler struct {
}

func (this *ClientHandler) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	fmt.Println("OnMessage", p)

	switch v := p.(type) {
	case *protocol.Packet:
		switch v.GetType() {
		case 1001:
			var rsp = &protocol.S2CJoinRoomRsp{}
			if err := v.UnmarshalProtoMessage(rsp); err != nil {
				return false
			}
			fmt.Println("加入房间返回结果", rsp.Code)
		}

	}
	return true
}

func (this *ClientHandler) OnClose(c *net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
