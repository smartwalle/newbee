package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee"
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
	//var h = &ServerHandler{}

	// 创建玩家信息
	var ps []*newbee.Player
	var p1 = newbee.NewPlayer(1001, "token1", 1)
	var p2 = newbee.NewPlayer(1002, "token2", 2)
	ps = append(ps, p1, p2)

	// 默认创建一个房间
	var rm = newbee.NewRoomManager()
	var r = rm.CreateRoom(ps)

	fmt.Println("房间创建成功，Id 为", r.GetId())

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		net4go.NewConn(c, p, rm)
	}
}

//
//type ServerHandler struct {
//}
//
//func (this *ServerHandler) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
//	fmt.Println("OnMessage", p)
//
//	switch v := p.(type) {
//	case *protocol.Packet:
//
//		switch v.GetType() {
//		case 1000:
//			var req = &protocol.JoinRoomReq{}
//			if err := v.UnmarshalProtoMessage(req); err != nil {
//				return false
//			}
//
//			fmt.Println(req.PlayerId, req.Token)
//
//			// 验证 Token
//			c.Set("player_id", req.PlayerId)
//
//			// 返回加入房间结果
//			var rsp = &protocol.JoinRoomRsp{}
//			rsp.Code = 1
//
//			c.WritePacket(protocol.NewPacket(1001, rsp))
//		case 1002:
//			var req = &protocol.LoadProgressReq{}
//			if err := v.UnmarshalProtoMessage(req); err != nil {
//				return false
//			}
//			fmt.Println("加入房间进度", c.Get("player_id"), req.Progress)
//		}
//	}
//	return true
//}
//
//func (this *ServerHandler) OnClose(c *net4go.Conn, err error) {
//	fmt.Println("OnClose", err)
//}
