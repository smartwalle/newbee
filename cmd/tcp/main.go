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

	for i := 0; i < 1000; i++ {
		c, err := net.Dial("tcp", ":9999")
		if err != nil {
			fmt.Println(err)
			return
		}

		var nConn = net4go.NewConn(c, p, h)
		nConn.Set("index", i)
	}

	select {}
}

type TCPHandler struct {
}

func (this *TCPHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			//fmt.Println(c.Get("index"), p.Message)
		case protocol.JoinRoomSuccess:
			go func(nConn net4go.Conn) {
				for {
					var p = &protocol.Packet{}
					p.Type = protocol.Heartbeat
					p.Message = "来自 TCP 的消息"

					nConn.WritePacket(p)

					time.Sleep(time.Millisecond * 66)
				}
			}(c)
		}
	}
	return true
}

func (this *TCPHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err, c.Get("index"))
}
