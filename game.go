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

	// OnJoinGame 有玩家加入会调用此方法
	OnJoinGame(player Player)

	// OnLeaveGame 有玩家离开会调用此方法
	OnLeaveGame(player Player)

	// OnCloseRoom 房间关闭的时候会调用此方法
	OnCloseRoom()

	// OnMessage 处理客户端消息
	OnMessage(player Player, packet net4go.Packet)

	// OnTick 定时器，Room 会定时调用，如果此方法返回 false，Room 将关闭
	OnTick(now int64) bool
}
