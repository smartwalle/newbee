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

	for i := 0; i < 1; i++ {
		c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8080/ws", nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		nConn := ws.NewConn(c, p, h)
		go func(nConn net4go.Conn) {
			for {
				var p = &protocol.Packet{}
				p.Type = protocol.Heartbeat
				p.Message = "来自 WS 的消息"

				nConn.AsyncWritePacket(p, 0)

				time.Sleep(time.Second * 5)
			}
		}(nConn)
	}

	select {}
}

type WSHandler struct {
}

func (this *WSHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			fmt.Println(p.Message)
		}
	}
	return true
}

func (this *WSHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err)
	os.Exit(-1)
}
