package newbee

import "github.com/smartwalle/net4go"

type GameState uint16

const (
	GameStatePending GameState = iota // 游戏等待开始
	GameStateGaming                   // 游戏进行中
	GameStateOver                     // 游戏结束
	GameStateStop                     // 游戏停止
)

type Game interface {
	// GetId 获取游戏 id
	GetId() uint64

	// RunInRoom Room 的 RunGame 方法会调用此方法
	RunInRoom(room Room)

	// Frequency 返回游戏的帧率
	Frequency() uint8

	// State 游戏状态
	State() GameState

	// OnMessage 处理客户端消息
	OnMessage(Player, net4go.Packet)

	// OnPlayerIn 有玩家加入
	OnPlayerIn(Player)

	// OnPlayerOut 有玩家离开
	OnPlayerOut(Player)

	// OnRoomClose 关闭房间
	OnRoomClose()

	// Tick 定时器，Room 会定时调用，如果此方法返回 false，Room 将关闭
	Tick(now int64) bool
}
