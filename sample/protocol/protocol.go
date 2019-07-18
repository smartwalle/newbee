package protocol

import (
	"encoding/binary"
	"github.com/smartwalle/net4go"
	"io"
)

type Protocol struct {
}

func (this *Protocol) Marshal(p net4go.Packet) []byte {
	var pData = p.Marshal()
	var data = make([]byte, 2+len(pData))
	binary.BigEndian.PutUint16(data[:2], uint16(len(pData)))
	copy(data[2:], pData)
	return data
}

func (this *Protocol) Unmarshal(r io.Reader) (net4go.Packet, error) {
	var lengthBytes = make([]byte, 2)
	if _, err := io.ReadFull(r, lengthBytes); err != nil {
		return nil, err
	}
	var length = binary.BigEndian.Uint16(lengthBytes)

	var buff = make([]byte, length)
	if _, err := io.ReadFull(r, buff); err != nil {
		return nil, err
	}

	var p = &Packet{}
	if err := p.Unmarshal(buff); err != nil {
		return nil, err
	}
	return p, nil
}
