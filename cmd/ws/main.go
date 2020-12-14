package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/net4go/ws"
	"github.com/smartwalle/newbee/cmd/protocol"
	"os"
	"time"
)

func main() {
	var p = &protocol.WSProtocol{}
	var h = &WSHandler{}

	for i := 0; i < 100; i++ {
		c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8080/ws", nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		ws.NewConn(c, ws.Text, p, h)
	}

	select {}
}

type WSHandler struct {
}

func (this *WSHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
		case protocol.JoinRoomSuccess:
			go func(nConn net4go.Conn) {
				for {
					var p = &protocol.Packet{}
					p.Type = protocol.Heartbeat
					p.Message = "来自 TCP 的消息"

					nConn.AsyncWritePacket(p)

					time.Sleep(time.Second * 10)
				}
			}(c)
		}
	}
	return true
}

func (this *WSHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err)
	os.Exit(-1)
}
