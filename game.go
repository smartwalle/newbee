package newbee

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"time"
)

type GameState uint16

const (
	GameStatePending GameState = iota // 游戏等待开始
	GameStateGaming                   // 游戏进行中
	GameStateOver                     // 游戏结束
	GameStateStop                     // 游戏停止
)

type Game struct {
	room *Room

	state                 GameState // 游戏状态
	createTime            int64     // 游戏创建时间
	startTime             int64     // 游戏正式开始时间
	maxPendingTime        int64     // 游戏准备阶段最长等待时间
	maxOfflinePendingTime int64     // 游戏所有玩家掉线最长等待时间
	offlineTime           int64     // 游戏所有玩家掉线开始时间点

	frameManager     *FrameManager
	clientFrameCount uint64 // 已经向客户端广播的帧数
}

func NewGame(room *Room) *Game {
	var g = &Game{}
	g.room = room
	g.state = GameStatePending
	g.createTime = time.Now().Unix()
	g.maxPendingTime = 20
	g.maxOfflinePendingTime = 20
	g.offlineTime = 0

	g.frameManager = NewFrameManager()
	return g
}

// Frequency 帧率，每秒钟发多少数据帧
func (this *Game) Frequency() uint8 {
	return 30
}

func (this *Game) State() GameState {
	return this.state
}

func (this *Game) ProcessMessage(player *Player, np net4go.Packet) {
	if p := np.(*protocol.Packet); p != nil {
		switch p.GetType() {
		case protocol.PT_HEARTBEAT:
			this.room.SendMessage(player.GetId(), protocol.NewPacket(protocol.PT_HEARTBEAT, nil))
			player.RefreshHeartbeatTime()
		case protocol.PT_LOADING_PROGRESS:
			if this.state != GameStatePending {
				return
			}

			var req = &protocol.C2SLoadingProgress{}
			if err := p.UnmarshalProtoMessage(req); err != nil {
				return
			}

			// 更新玩家的加载进度
			player.UpdateLoadProgress(req.Progress)

			// 向所有玩家广播加载进度
			var rsp = &protocol.S2CLoadingProgress{}
			for _, player := range this.room.GetPlayers() {
				rsp.Infos = append(rsp.Infos, &protocol.LoadingProgressInfo{PlayerId: player.GetId(), Progress: player.GetLoadingProgress()})
			}
			this.room.Broadcast(protocol.NewPacket(protocol.PT_LOADING_PROGRESS, rsp))
		case protocol.PT_GAME_READY:
			if this.state != GameStatePending {
				return
			}
			player.Ready()
		case protocol.PT_GAME_FRAME:
			if this.state != GameStateGaming {
				return
			}
			var req = &protocol.C2SGameFrame{}
			if err := p.UnmarshalProtoMessage(req); err != nil {
				return
			}

			var cmd = &protocol.FrameCommand{}
			cmd.PlayerId = player.GetId()
			cmd.PlayerMove = req.PlayerMove

			this.frameManager.PushFrame(req.FrameId, cmd)
		}
	}
}

// GameStart 开始游戏
func (this *Game) GameStart() {
	this.state = GameStateGaming

	var rsp = &protocol.S2CGameReady{}
	this.room.Broadcast(protocol.NewPacket(protocol.PT_GAME_READY, rsp))
}

// GameOver 结束游戏
func (this *Game) GameOver() {
	this.state = GameStateOver
	// TODO 判断是否为正常游戏结束
	// TODO 向所有玩家发送游戏结束指令，包含游戏结果数据
	// 发送游戏结束指令之后，将玩家标记为未准备状态

	fmt.Println("Game Over")
}

// GameStop 停止游戏
func (this *Game) GameStop() {
	this.state = GameStateStop
	// TODO 向所有玩家发送游戏停止的指令

	fmt.Println("Game Stop")
}

func (this *Game) CheckOver() bool {

	return true
}

func (this *Game) Tick(now int64) bool {
	switch this.state {
	case GameStatePending:
		// 游戏等待开始
		var pendingTime = now - this.createTime
		if pendingTime < this.maxPendingTime {
			// 如果等待时间小于最大等待时间，则检查所有的玩家是否都已准备就绪
			// 如果所有的玩家都已准备就绪，则开始游戏
			if this.room.CheckAllPlayerReady() {
				this.GameStart()
			}
		} else {
			// 如果等待时间大于等于最大等待时间
			if this.room.GetReadyPlayerCount() > 0 {
				// 如果已准备就绪玩家数量大于 0，则直接开始游戏
				this.GameStart()
			} else {
				// 如果没有准备就绪的玩家，则直接结束游戏
				this.GameOver()
			}
		}
		return true
	case GameStateGaming:
		// 游戏进行中

		// 所有玩家都掉线超时一定时间就判断游戏停止
		var online = this.room.GetOnlinePlayerCount()
		if online <= 0 {
			if this.offlineTime <= 0 {
				this.offlineTime = now
			}

			var pendingTime = now - this.offlineTime
			if pendingTime >= this.maxOfflinePendingTime {
				this.GameOver()
				return false
			}
		} else {
			this.offlineTime = 0
		}

		this.frameManager.Tick()

		this.broadcastFrame()
		return true
	case GameStateOver:
		// 游戏结束 - 现在结束之后直接停止
		this.GameStop()
		return true
	case GameStateStop:
		// 游戏停止
		return false
	}
	return false
}

func (this *Game) broadcastFrame() {
	var frameCount = this.frameManager.GetFrameCount()

	defer func() {
		this.clientFrameCount = frameCount
	}()

	var rsp = &protocol.S2CGameFrame{}
	for i := this.clientFrameCount; i < frameCount; i++ {
		var frame = this.frameManager.GetFrame(i)

		if frame == nil && i != frameCount-1 {
			continue
		}

		rsp.Frames = append(rsp.Frames, frame)
	}

	if len(rsp.Frames) > 0 {
		this.room.Broadcast(protocol.NewPacket(protocol.PT_GAME_FRAME, rsp))
	}
}
