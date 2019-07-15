package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":6655")
	if err != nil {
		fmt.Println(err)
		return
	}

	var p = &protocol.Protocol{}
	var h = &ServerHandler{}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		net4go.NewConn(c, p, h)
	}
}

type ServerHandler struct {
}

func (this *ServerHandler) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	fmt.Println("OnMessage", p)

	switch v := p.(type) {
	case *protocol.Packet:

		switch v.GetType() {
		case 1000:
			var req = &protocol.C2SJoinRoomReq{}
			if err := v.UnmarshalProtoMessage(req); err != nil {
				return false
			}

			fmt.Println(req.PlayerId, req.Token)

			// 验证 Token
			c.Set("player_id", req.PlayerId)

			var rsp = &protocol.S2CJoinRoomRsp{}
			rsp.Code = 10

			c.WritePacket(protocol.NewPacket(1001, rsp))

		}
	}
	return true
}

func (this *ServerHandler) OnClose(c *net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
