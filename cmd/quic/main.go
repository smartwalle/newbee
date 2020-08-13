package main

import (
	"crypto/tls"
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/net4go/quic"
	"github.com/smartwalle/newbee/cmd/protocol"
	"os"
	"time"
)

func main() {
	var p = &protocol.TCPProtocol{}
	var h = &QUICHandler{}

	for i := 0; i < 100; i++ {
		c, err := quic.Dial("127.0.0.1:8898", &tls.Config{InsecureSkipVerify: true,
			NextProtos: []string{"newbee"}}, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		nConn := net4go.NewConn(c, p, h)

		go func(nConn net4go.Conn) {
			for {
				var p = &protocol.Packet{}
				p.Type = protocol.Heartbeat
				p.Message = "来自 QUIC 的消息"

				nConn.AsyncWritePacket(p, 0)

				time.Sleep(time.Millisecond * 10)
			}
		}(nConn)
	}

	select {}
}

type QUICHandler struct {
}

func (this *QUICHandler) OnMessage(c net4go.Conn, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			fmt.Println(p.Message)
		}
	}
	return true
}

func (this *QUICHandler) OnClose(c net4go.Conn, err error) {
	fmt.Println("OnClose", err)
	os.Exit(-1)
}
