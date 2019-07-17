package newbee

import (
	"fmt"
	"github.com/smartwalle/newbee/protocol"
)

type Frame struct {
	Id   uint64
	Data []*protocol.FrameData
}

func NewFrame(id uint64) *Frame {
	return &Frame{Id: id}
}

type LockStep struct {
	frameCount uint64
	frames     map[uint64]*Frame
}

func NewLockStep() *LockStep {
	var ls = &LockStep{}
	ls.frameCount = 0
	ls.frames = make(map[uint64]*Frame)
	return ls
}

func (this *LockStep) Push(frameId uint64, data *protocol.FrameData) {
	if frameId != this.frameCount {
		fmt.Println(frameId, this.frameCount)
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

func (this *LockStep) Tick() uint64 {
	this.frameCount++
	return this.frameCount
}

func (this *LockStep) GetFrameCount() uint64 {
	return this.frameCount
}

func (this *LockStep) GetFrame(id uint64) *Frame {
	return this.frames[id]
}
