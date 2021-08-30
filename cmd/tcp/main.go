package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/cmd/protocol"
	"net"
	"time"
)

func main() {
	var p = &protocol.TCPProtocol{}
	var h = &TCPHandler{}

	for i := 0; i < 100; i++ {
		c, err := net.Dial("tcp", ":9999")
		if err != nil {
			fmt.Println(err)
			return
		}

		var nSess = net4go.NewSession(c, p, h)
		nSess.Set("index", i)
	}

	select {}
}

type TCPHandler struct {
}

func (this *TCPHandler) OnMessage(sess net4go.Session, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			fmt.Println(sess.Get("index"), p.Message)
		case protocol.JoinRoomSuccess:
			go func(nSess net4go.Session) {
				for {
					var p = &protocol.Packet{}
					p.Type = protocol.Heartbeat
					p.Message = "来自 TCP 的消息"

					nSess.WritePacket(p)

					time.Sleep(time.Second * 1)
				}
			}(sess)
		}
	}
	return true
}

func (this *TCPHandler) OnClose(sess net4go.Session, err error) {
	fmt.Println("OnClose", err, sess.Get("index"))
}
