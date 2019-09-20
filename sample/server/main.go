package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee"
	"github.com/smartwalle/newbee/sample/protocol"
	"github.com/smartwalle/newbee/sample/server/game1"
	"github.com/smartwalle/newbee/sample/server/room"
	"net"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	l, err := net.Listen("tcp", "192.168.1.99:6666")
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		http.ListenAndServe(":6661", nil)
	}()

	var p = &protocol.Protocol{}
	var rm = room.NewRoomManager()

	var numberOfRoom = 200

	for roomId := 0; roomId < numberOfRoom; roomId++ {

		var playerList = make([]newbee.Player, 0, 10)
		var game = game1.NewGame(uint64(roomId))

		for playerIndex := 0; playerIndex < 10; playerIndex++ {
			var playerId = uint64(roomId*10+playerIndex) + 1

			var player = newbee.NewPlayer(playerId, newbee.WithPlayerToken("token"), newbee.WithPlayerGroup(1), newbee.WithPlayerIndex(uint32(playerIndex)))
			playerList = append(playerList, player)

			fmt.Println(playerId)
		}
		var room = rm.CreateRoom(uint64(roomId), playerList, game)
		fmt.Println("房间创建成功，Id 为", room.GetId())
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		net4go.NewConn(c, p, rm)
	}
}
