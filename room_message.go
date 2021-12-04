package newbee

func (this *room) onMessage(game Game, playerId int64, data interface{}) {
	var p = this.GetPlayer(playerId)
	if p == nil {
		return
	}
	game.OnMessage(p, data)
}

func (this *room) onDequeue(game Game, data interface{}) {
	game.OnDequeue(data)
}

func (this *room) onJoinRoom(game Game, playerId int64) {
	var p = this.GetPlayer(playerId)
	if p == nil {
		return
	}
	game.OnJoinRoom(p)
}

func (this *room) onLeaveRoom(game Game, playerId int64, err error) {
	var p = this.popPlayer(playerId)
	if p == nil {
		return
	}
	game.OnLeaveRoom(p, err)
}
