package newbee

import (
	"fmt"
	"github.com/smartwalle/net4go"
	"github.com/smartwalle/newbee/protocol"
	"math/rand"
	"sync"
)

type RoomManager struct {
	mu    sync.RWMutex
	rooms map[uint64]*Room
}

func NewRoomManager() *RoomManager {
	var rm = &RoomManager{}
	rm.rooms = make(map[uint64]*Room)
	return rm
}

func (this *RoomManager) CreateRoom(players []*Player) *Room {
	this.mu.Lock()
	defer this.mu.Unlock()

	var r = newRoom(players)
	// TODO 房间 id 生成规则
	r.id = rand.Uint64()

	this.rooms[r.id] = r

	go r.run()

	return r
}

func (this *RoomManager) GetRoom(roomId uint64) *Room {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var r = this.rooms[roomId]
	return r
}

// --------------------------------------------------------------------------------
func (this *RoomManager) OnMessage(c *net4go.Conn, p net4go.Packet) bool {
	switch v := p.(type) {
	case *protocol.Packet:
		switch v.GetType() {
		case protocol.PT_JOIN_ROOM_REQ:
			var req = &protocol.JoinRoomReq{}
			if err := v.UnmarshalProtoMessage(req); err != nil {
				return false
			}
			return this.joinRoom(c, req)
		default:
			c.Close()
		}
	default:
		c.Close()
	}
	return true
}

func (this *RoomManager) OnClose(c *net4go.Conn, err error) {
	fmt.Println("OnClose", err)
}

// --------------------------------------------------------------------------------
func (this *RoomManager) joinRoom(c *net4go.Conn, req *protocol.JoinRoomReq) bool {
	fmt.Println(req.RoomId, req.PlayerId, req.Token)

	// 验证要加入的房间是否存在
	var room = this.GetRoom(req.RoomId)
	if room == nil {
		this.joinRoomRsp(c, protocol.JOIN_ROOM_CODE_ROOM_NOT_EXIST)
		return false
	}

	// 验证房间是否有该用户的信息及该用户是否已经有连接
	var player = room.GetPlayer(req.PlayerId)
	if player == nil || player.IsOnline() {
		this.joinRoomRsp(c, protocol.JOIN_ROOM_CODE_PLAYER_NOT_EXIST)
		return false
	}

	// 验证用户的 Token 信息
	if player.GetToken() != req.Token {
		this.joinRoomRsp(c, protocol.JOIN_ROOM_CODE_TOKEN_NOT_EXIST)
		return false
	}

	room.Join(player.GetId(), c)

	this.joinRoomRsp(c, protocol.JOIN_ROOM_CODE_SUCCESS)
	return true
}

func (this *RoomManager) joinRoomRsp(c *net4go.Conn, code protocol.JOIN_ROOM_CODE) {
	var rsp = &protocol.JoinRoomRsp{}
	rsp.Code = code
	c.WritePacket(protocol.NewPacket(protocol.PT_JOIN_ROOM_RSP, rsp))
}
