package protocol

type PacketType uint16

const (
	Heartbeat       PacketType = 1
	JoinRoomSuccess PacketType = 2
)

type Packet struct {
	Type    PacketType `json:"type"`
	Message string     `json:"message"`
	X       int        `json:"x"`
	Y       int        `json:"y"`
}

func (this *Packet) MarshalPacket() ([]byte, error) {
	//var data = make([]byte, 2+len(this.Message))
	//binary.BigEndian.PutUint16(data[0:2], uint16(this.Type))
	//copy(data[2:], []byte(this.Message))
	//return data, nil
	return nil, nil
}

func (this *Packet) UnmarshalPacket(data []byte) error {
	//this.Type = PacketType(binary.BigEndian.Uint16(data[:2]))
	//this.Message = string(data[2:])
	return nil
}
