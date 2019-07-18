package game1

import (
	"github.com/smartwalle/newbee/sample/protocol"
)

type FrameManager struct {
	frameCount uint64
	frames     map[uint64]*protocol.GameFrame
}

func NewFrameManager() *FrameManager {
	var ls = &FrameManager{}
	ls.frameCount = 0
	ls.frames = make(map[uint64]*protocol.GameFrame)
	return ls
}

func (this *FrameManager) Reset() {
	this.frameCount = 0
	this.frames = make(map[uint64]*protocol.GameFrame)
}

func (this *FrameManager) PushFrame(frameId uint64, cmd *protocol.FrameCommand) {
	if frameId != this.frameCount {
		return
	}

	frame, ok := this.frames[this.frameCount]
	if ok == false {
		frame = &protocol.GameFrame{}
		frame.FrameId = this.frameCount
		this.frames[this.frameCount] = frame
	}

	for _, d := range frame.Commands {
		if d.PlayerId == cmd.PlayerId {
			return
		}
	}

	frame.Commands = append(frame.Commands, cmd)
}

func (this *FrameManager) Tick() uint64 {
	this.frameCount++
	return this.frameCount
}

func (this *FrameManager) GetFrameCount() uint64 {
	return this.frameCount
}

func (this *FrameManager) GetFrame(id uint64) *protocol.GameFrame {
	return this.frames[id]
}
