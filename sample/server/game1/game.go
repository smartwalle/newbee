package game1

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee"
	"github.com/smartwalle/newbee/sample/protocol"
	"time"
)

type game struct {
	id uint64

	room newbee.Room

	state                 newbee.GameState // 游戏状态
	createTime            int64            // 游戏创建时间
	startTime             int64            // 游戏正式开始时间
	maxPendingTime        int64            // 游戏准备阶段最长等待时间
	maxOfflinePendingTime int64            // 游戏所有玩家掉线最长等待时间
	offlineTime           int64            // 游戏所有玩家掉线开始时间点

	frameManager     *FrameManager // 游戏帧管理器
	clientFrameCount uint64        // 已经向客户端广播的帧数
}

func NewGame(id uint64) newbee.Game {
	var g = &game{}
	g.id = id
	g.state = newbee.GameStatePending
	g.createTime = time.Now().Unix()
	g.maxPendingTime = 10
	g.maxOfflinePendingTime = 20
	g.offlineTime = 0

	g.frameManager = NewFrameManager()
	return g
}

func (this *game) GetId() uint64 {
	return this.id
}

func (this *game) RunInRoom(room newbee.Room) {
	this.room = room
}

// Frequency 帧率，每秒钟发多少数据帧
func (this *game) Frequency() uint8 {
	return 30
}

func (this *game) State() newbee.GameState {
	return this.state
}

func (this *game) OnJoinGame(p newbee.Player) {
	fmt.Println("OnJoinGame")
}

func (this *game) OnLeaveGame(p newbee.Player) {
	fmt.Println("OnLeaveGame")
}

func (this *game) OnCloseRoom() {
	fmt.Println("OnCloseRoom")
}

func (this *game) OnMessage(player newbee.Player, np net4go.Packet) {
	if p := np.(*protocol.Packet); p != nil {
		switch p.GetType() {
		case protocol.PT_HEARTBEAT:
			this.SendPacket(player.GetId(), protocol.NewPacket(protocol.PT_HEARTBEAT, nil))
			player.RefreshHeartbeatTime()
		case protocol.PT_LOADING_PROGRESS:
			if this.state != newbee.GameStatePending {
				return
			}

			var req = &protocol.C2SLoadingProgress{}
			if err := p.UnmarshalProtoMessage(req); err != nil {
				return
			}

			// 更新玩家的加载进度
			player.UpdateLoadingProgress(req.Progress)

			// 向所有玩家广播加载进度
			var rsp = &protocol.S2CLoadingProgress{}
			for _, player := range this.room.GetPlayers() {
				rsp.Infos = append(rsp.Infos, &protocol.LoadingProgressInfo{PlayerId: player.GetId(), Progress: player.GetLoadingProgress()})
			}
			this.BroadcastPacket(protocol.NewPacket(protocol.PT_LOADING_PROGRESS, rsp))
		case protocol.PT_GAME_READY:
			if this.state != newbee.GameStatePending {
				return
			}
			player.Ready()
		case protocol.PT_GAME_FRAME:
			if this.state != newbee.GameStateGaming {
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
func (this *game) GameStart() {
	this.state = newbee.GameStateGaming

	var rsp = &protocol.S2CGameReady{}
	this.BroadcastPacket(protocol.NewPacket(protocol.PT_GAME_READY, rsp))
}

// GameOver 结束游戏
func (this *game) GameOver() {
	this.state = newbee.GameStateOver
	// TODO 判断是否为正常游戏结束
	// TODO 向所有玩家发送游戏结束指令，包含游戏结果数据
	// 发送游戏结束指令之后，将玩家标记为未准备状态

	fmt.Println("game Over")
}

// GameStop 停止游戏
func (this *game) GameStop() {
	this.state = newbee.GameStateStop
	// TODO 向所有玩家发送游戏停止的指令

	fmt.Println("game Stop")
}

func (this *game) CheckOver() bool {

	return true
}

func (this *game) OnTick(now int64) bool {
	switch this.state {
	case newbee.GameStatePending:
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
	case newbee.GameStateGaming:
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
	case newbee.GameStateOver:
		// 游戏结束 - 现在结束之后直接停止
		this.GameStop()
		return true
	case newbee.GameStateStop:
		// 游戏停止
		return false
	}
	return false
}

func (this *game) broadcastFrame() {
	var frameCount = this.frameManager.GetFrameCount()

	defer func() {
		this.clientFrameCount = frameCount
	}()

	var rsp = &protocol.S2CGameFrame{}
	for i := this.clientFrameCount; i < frameCount; i++ {
		var frame = this.frameManager.GetFrame(i)

		//if frame == nil && i != frameCount-1 {
		//	continue
		//}

		// 如果该帧没有数据，则构造一帧空数据
		if frame == nil {
			frame = &protocol.GameFrame{}
			frame.FrameId = i
		}

		rsp.Frames = append(rsp.Frames, frame)
	}

	if len(rsp.Frames) > 0 {
		this.BroadcastPacket(protocol.NewPacket(protocol.PT_GAME_FRAME, rsp))
	}
}

func (this *game) SendPacket(playerId uint64, p *protocol.Packet) {
	if p != nil {
		this.room.SendPacket(playerId, p)
	}
}

func (this *game) BroadcastPacket(p *protocol.Packet) {
	if p != nil {
		this.room.BroadcastPacket(p)
	}
}

func (this *game) cleanup() {
	this.room = nil
	this.frameManager = nil
	this.clientFrameCount = 0
}
