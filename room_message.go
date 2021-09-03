package newbee

import "github.com/smartwalle/net4go"

func (this *room) onMessage(game Game, playerId int64, data interface{}) (exit bool) {
	var p = this.GetPlayer(playerId)
	if p == nil {
		return true
	}
	game.OnMessage(p, data)
	return false
}

func (this *room) onJoinRoom(game Game, playerId int64, sess net4go.Session) (exit bool) {
	var p = this.GetPlayer(playerId)
	if p == nil {
		return true
	}
	p.Connect(sess)
	game.OnJoinRoom(p)
	return false
}

func (this *room) onLeaveRoom(game Game, playerId int64) (exit bool) {
	var p = this.popPlayer(playerId)
	if p == nil {
		return true
	}
	game.OnLeaveRoom(p)
	p.Close()
	return false
}
