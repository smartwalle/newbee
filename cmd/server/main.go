package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/net4go/quic"
	"github.com/smartwalle/net4go/ws"
	"github.com/smartwalle/newbee"
	"github.com/smartwalle/newbee/cmd/protocol"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"
)

func main() {
	var tcpp = &protocol.TCPProtocol{}
	var wsp = &protocol.WSProtocol{}

	var room = newbee.NewRoom(100, "xxx", nil)

	var game = &Game{}
	go room.Run(game)

	var mu = &sync.Mutex{}
	var playerId uint64 = 0

	// ws
	go func() {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			var c, err = upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			nConn := ws.NewConn(c, wsp, nil)

			mu.Lock()
			playerId = playerId + 1
			room.AddPlayer(newbee.NewPlayer(playerId), nConn)
			mu.Unlock()
		})
		http.ListenAndServe(":8080", nil)
	}()

	// tcp
	go func() {
		l, err := net.Listen("tcp", "127.0.0.1:8899")
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			c, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}

			nConn := net4go.NewConn(c, tcpp, nil)

			mu.Lock()
			playerId = playerId + 1
			room.AddPlayer(newbee.NewPlayer(playerId), nConn)
			mu.Unlock()
		}
	}()

	// quic
	go func() {
		l, err := quic.Listen("127.0.0.1:8898", generateTLSConfig(), nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			c, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}

			nConn := net4go.NewConn(c, tcpp, nil)

			mu.Lock()
			playerId = playerId + 1
			room.AddPlayer(newbee.NewPlayer(playerId), nConn)
			mu.Unlock()
		}
	}()

	select {}
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"newbee"},
	}
}

type Game struct {
	id    uint64
	room  newbee.Room
	state newbee.GameState
}

func (this *Game) GetId() uint64 {
	return this.id
}

func (this *Game) GetState() newbee.GameState {
	return this.state
}

func (this *Game) TickInterval() time.Duration {
	return 0
}

func (this *Game) OnTick(now int64) bool {
	fmt.Println("OnTick", now)
	return true
}

func (this *Game) OnMessage(player newbee.Player, packet net4go.Packet) {
	if p := packet.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			fmt.Println(player.GetId(), p.Message)
			p.Message = "来自服务器的消息"
			player.SendPacket(p)
		}
	}
}

func (this *Game) OnRunInRoom(room newbee.Room) {
	this.room = room
}

func (this *Game) OnJoinRoom(player newbee.Player) {
	fmt.Println("OnJoinRoom", player.GetId())
}

func (this *Game) OnLeaveRoom(player newbee.Player) {
	fmt.Println("OnLeaveRoom", player.GetId())
}

func (this *Game) OnCloseRoom(room newbee.Room) {
	fmt.Println("OnCloseRoom")
}
