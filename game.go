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
	// RunInRoom Room 的 RunGame 方法会调用此方法
	RunInRoom(room Room)

	// Frequency 返回游戏的帧率
	Frequency() uint8

	// State 游戏状态
	State() GameState

	// OnMessage 处理客户端消息
	OnMessage(Player, net4go.Packet)

	// Tick 定时器，Room 会定时调用
	Tick(now int64) bool
}
