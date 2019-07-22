package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee"
	"github.com/smartwalle/newbee/sample/protocol"
	"github.com/smartwalle/newbee/sample/server/game1"
	"github.com/smartwalle/newbee/sample/server/room"
	"net"
)

func main() {
	l, err := net.Listen("tcp", "192.168.1.99:6666")
	if err != nil {
		fmt.Println(err)
		return
	}

	var p = &protocol.Protocol{}
	//var h = &ServerHandler{}

	// 创建玩家信息
	var ps []newbee.Player
	var p1 = newbee.NewPlayer(1001, "token1", 1)
	var p2 = newbee.NewPlayer(1002, "token2", 2)
	ps = append(ps, p1, p2)

	// 创建游戏信息
	var game = game1.NewGame(123)

	// 默认创建一个房间
	var rm = room.NewRoomManager()
	var r = rm.CreateRoom(ps, game)

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
