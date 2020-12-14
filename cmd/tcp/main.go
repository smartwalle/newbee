package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/cmd/protocol"
	"net"
	"os"
	"time"
)

func main() {
	var p = &protocol.TCPProtocol{}
	var h = &TCPHandler{}

	for i := 0; i < 100; i++ {
		c, err := net.Dial("tcp", "127.0.0.1:8899")
		if err != nil {
			fmt.Println(err)
			return
		}

		net4go.NewConn(c, p, h)
	}

	select {}
}

type TCPHandler struct {
}

func (this *TCPHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
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

func (this *TCPHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err, c.Get("index"))
	os.Exit(-1)
}
