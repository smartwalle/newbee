package protocol

import "encoding/binary"

type PacketType uint16

const (
	Heartbeat PacketType = iota + 1
)

type Packet struct {
	Type    PacketType `json:"type"`
	Message string     `json:"message"`
}

func (this *Packet) MarshalPacket() ([]byte, error) {
	var data = make([]byte, 2+len(this.Message))
	binary.BigEndian.PutUint16(data[0:2], uint16(this.Type))
	copy(data[2:], []byte(this.Message))
	return data, nil
}

func (this *Packet) UnmarshalPacket(data []byte) error {
	this.Type = PacketType(binary.BigEndian.Uint16(data[:2]))
	this.Message = string(data[2:])
	return nil
}
