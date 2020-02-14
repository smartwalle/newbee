package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/sample/protocol"
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

		nConn := net4go.NewWsConn(c, p, h)
		go func(nConn net4go.Conn) {
			for {
				var p = &protocol.Packet{}
				p.Type = protocol.Heartbeat
				p.Message = "来处 WS 的消息"

				nConn.AsyncWritePacket(p, 0)

				time.Sleep(time.Millisecond * 10)
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
