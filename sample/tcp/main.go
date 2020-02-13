package main

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/sample/protocol"
	"net"
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

		nConn := net4go.NewConn(c, p, h)
		go func(nConn net4go.Conn) {
			for {
				var p = &protocol.Packet{}
				p.Type = protocol.Heartbeat
				p.Message = "来自 TCP 的消息"

				nConn.AsyncWritePacket(p, 0)

				time.Sleep(time.Second * 1)
			}
		}(nConn)
	}

	select {}
}

type TCPHandler struct {
}

func (this *TCPHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			fmt.Println(p.Message)
		}
	}
	return true
}

func (this *TCPHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}
