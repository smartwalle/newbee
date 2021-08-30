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

		ws.NewSession(c, ws.Text, p, h)
	}

	select {}
}

type WSHandler struct {
}

func (this *WSHandler) OnMessage(sess net4go.Session, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
		case protocol.JoinRoomSuccess:
			go func(nSess net4go.Session) {
				for {
					var p = &protocol.Packet{}
					p.Type = protocol.Heartbeat
					p.Message = "来自 TCP 的消息"

					nSess.AsyncWritePacket(p)

					time.Sleep(time.Second * 1)
				}
			}(sess)
		}
	}
	return true
}

func (this *WSHandler) OnClose(sess net4go.Session, err error) {
	fmt.Println("OnClose", err)
	os.Exit(-1)
}
