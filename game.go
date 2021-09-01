package newbee

import (
	"time"
)

type GameState uint16

const (
	GameStatePending GameState = iota // 游戏等待开始
	GameStateGaming                   // 游戏进行中
	GameStateOver                     // 游戏结束
	GameStateStop                     // 游戏停止
)

type Game interface {
	// GetId 获取游戏 id
	GetId() int64

	// GetState 游戏状态
	GetState() GameState

	// TickInterval 返回刷新时间间隔，Room 将按照该时间间隔调用 OnTick() 方法，返回小于等于 0 的时候，将禁用定时刷新
	TickInterval() time.Duration

	// OnTick 定时器，Room 会定时调用，如果此方法返回 false，Room 将关闭
	OnTick()

	// OnMessage 处理客户端消息
	OnMessage(player Player, message interface{})

	// OnDequeue 处理自定义消息
	OnDequeue(message interface{})

	// OnRunInRoom Room 启动成功之后会调用此方法
	OnRunInRoom(room Room)

	// OnJoinRoom 玩家建立网络连接会调用此方法
	OnJoinRoom(player Player)

	// OnLeaveRoom 玩家断开网络连接会调用此方法
	OnLeaveRoom(player Player)

	// OnCloseRoom 房间关闭的时候会调用此方法
	OnCloseRoom(room Room)
}
