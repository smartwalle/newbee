package newbee

import (
	"github.com/smartwalle/net4go"
)

type GameState uint16

const (
	GameStatePending GameState = iota // 游戏等待开始
	GameStateGaming                   // 游戏进行中
	GameStateOver                     // 游戏结束
	GameStateStop                     // 游戏停止
)

type Game interface {
	RunInRoom(room Room)

	Frequency() uint8

	State() GameState

	ProcessMessage(player Player, np net4go.Packet)

	Tick(now int64) bool
}
