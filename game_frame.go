package newbee

import (
	"github.com/smartwalle/newbee/protocol"
)

type Frame struct {
	Id   uint64
	Data []*protocol.FrameData
}

func NewFrame(id uint64) *Frame {
	return &Frame{Id: id}
}

type FrameManager struct {
	frameCount uint64
	frames     map[uint64]*Frame
}

func NewLockStep() *FrameManager {
	var ls = &FrameManager{}
	ls.frameCount = 0
	ls.frames = make(map[uint64]*Frame)
	return ls
}

func (this *FrameManager) Reset() {
	this.frameCount = 0
	this.frames = make(map[uint64]*Frame)
}

func (this *FrameManager) Push(frameId uint64, data *protocol.FrameData) {
	if frameId != this.frameCount {
		return
	}

	frame, ok := this.frames[this.frameCount]
	if ok == false {
		frame = NewFrame(this.frameCount)
		this.frames[this.frameCount] = frame
	}

	for _, d := range frame.Data {
		if d.PlayerId == data.PlayerId {
			return
		}
	}

	frame.Data = append(frame.Data, data)
}

func (this *FrameManager) Tick() uint64 {
	this.frameCount++
	return this.frameCount
}

func (this *FrameManager) GetFrameCount() uint64 {
	return this.frameCount
}

func (this *FrameManager) GetFrame(id uint64) *Frame {
	return this.frames[id]
}
