package protocol

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
)

type Packet struct {
	pType uint32
	data  []byte
}

func (this *Packet) Marshal() []byte {
	var dataLen = len(this.data)
	var data = make([]byte, 4+dataLen)
	binary.BigEndian.PutUint32(data[:4], this.pType)
	if dataLen > 0 {
		copy(data[4:], this.data)
	}
	return data
}

func (this *Packet) Unmarshal(data []byte) error {
	this.pType = binary.BigEndian.Uint32(data[:4])
	this.data = data[4:]
	return nil
}

func (this *Packet) GetType() PT {
	return PT(this.pType)
}

func (this *Packet) GetData() []byte {
	return this.data
}

func (this *Packet) UnmarshalProtoMessage(obj proto.Message) error {
	return proto.Unmarshal(this.data, obj)
}

func NewPacket(pType PT, obj interface{}) *Packet {
	var p = &Packet{}
	p.pType = uint32(pType)

	switch v := obj.(type) {
	case []byte:
		p.data = v
	case proto.Message:
		mData, err := proto.Marshal(v)
		if err != nil {
			// TODO 处理错误
			return nil
		}
		p.data = mData
	case nil:
	default:
		// TODO 处理类型不存在的错误
		return nil
	}

	return p
}
