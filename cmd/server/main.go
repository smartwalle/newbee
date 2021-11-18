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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	var tcpp = &protocol.TCPProtocol{}
	var wsp = &protocol.WSProtocol{}

	var waiter = &sync.WaitGroup{}

	var room = newbee.NewRoom(100, newbee.WithWaiter(waiter), newbee.WithFrame())

	var game = &Game{}
	go func() {
		fmt.Println("开始游戏...")

		var err = room.Run(game)

		if err != nil {
			fmt.Println("游戏异常结束:", err)
		} else {
			fmt.Println("游戏结束.")
		}
	}()

	// sleep 一会儿，让 Room 运行 Game
	time.Sleep(time.Second * 1)

	var mu = &sync.Mutex{}
	var playerId int64 = 0

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
			nSess := ws.NewSession(c, ws.Text, wsp, nil)

			mu.Lock()
			playerId = playerId + 1
			fmt.Println(room.AddPlayer(newbee.NewPlayer(playerId, nSess)))
			mu.Unlock()
		})
		http.ListenAndServe(":8080", nil)
	}()

	// tcp
	go func() {
		l, err := net.Listen("tcp", ":9999")
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

			nSess := net4go.NewSession(c, tcpp, nil, net4go.WithNoDelay(false))

			mu.Lock()
			playerId = playerId + 1
			room.AddPlayer(newbee.NewPlayer(playerId, nSess))
			mu.Unlock()
		}
	}()

	// quic
	go func() {
		l, err := quic.Listen(":8898", generateTLSConfig(), nil)
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

			nSess := net4go.NewSession(c, tcpp, nil)

			mu.Lock()
			playerId = playerId + 1
			fmt.Println(room.AddPlayer(newbee.NewPlayer(playerId, nSess)))
			mu.Unlock()
		}
	}()

	fmt.Println("运行中...")

	var c = make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
MainLoop:
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			break MainLoop
		}
	}

	fmt.Println("开始关闭游戏.")
	room.Close()
	fmt.Println("关闭中...")
	waiter.Wait()
	fmt.Println("结束.")
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
	id        int64
	room      newbee.Room
	state     newbee.GameState
	tickCount int64
}

func (this *Game) GetId() int64 {
	return this.id
}

func (this *Game) GetState() newbee.GameState {
	return this.state
}

func (this *Game) TickInterval() time.Duration {
	return time.Second / 100
}

func (this *Game) OnTick() {
	this.tickCount++
	//fmt.Println("OnTick", time.Now(), this.tickCount)
}

func (this *Game) OnMessage(player newbee.Player, message interface{}) {
	if p := message.(*protocol.Packet); p != nil {
		switch p.Type {
		case protocol.Heartbeat:
			p.Message = "来自服务器的消息"
			player.AsyncSendPacket(p)

			this.room.Enqueue(fmt.Sprintf("%s haha %d", time.Now(), player.GetId()))
		}
	}
}

func (this *Game) OnDequeue(message interface{}) {
	fmt.Println(message)
}

func (this *Game) OnRunInRoom(room newbee.Room) {
	this.room = room
}

func (this *Game) OnJoinRoom(player newbee.Player) {
	fmt.Println("OnJoinRoom", player.GetId())

	var p = &protocol.Packet{}
	p.Type = protocol.JoinRoomSuccess
	player.AsyncSendPacket(p)

}

func (this *Game) OnLeaveRoom(player newbee.Player) {
	fmt.Println("OnLeaveRoom", player.GetId(), this.room.GetState())
	fmt.Println("保存玩家数据:", player.GetId())
	time.Sleep(time.Second * 3)
	fmt.Println("保存玩家数据完成:", player.GetId())
}

func (this *Game) OnCloseRoom(room newbee.Room) {
	fmt.Println("OnCloseRoom", room.GetPlayerCount())
}
