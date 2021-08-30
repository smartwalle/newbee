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

		net4go.NewSession(c, p, h)
	}

	select {}
}

type QUICHandler struct {
}

func (this *QUICHandler) OnMessage(sess net4go.Session, packet net4go.Packet) bool {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
		case protocol.JoinRoomSuccess:
			go func(nSess net4go.Session) {
				for {
					var p = &protocol.Packet{}
					p.Type = protocol.Heartbeat
					p.Message = "来自 QUIC 的消息"

					nSess.AsyncWritePacket(p)

					time.Sleep(time.Second * 10)
				}
			}(sess)
		}
	}
	return true
}

func (this *QUICHandler) OnClose(sess net4go.Session, err error) {
	fmt.Println("OnClose", err)
	os.Exit(-1)
}
