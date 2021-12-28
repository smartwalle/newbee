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

func (this *room) onJoinRoom(game Game, player Player) {
	if player == nil {
		return
	}
	this.mu.Lock()

	if _, ok := this.players[player.GetId()]; ok {
		this.mu.Unlock()
		return
	}

	if player.Connected() {
		this.players[player.GetId()] = player

		var sess = player.Session()
		sess.SetId(player.GetId())
		sess.UpdateHandler(this)
	}
	this.mu.Unlock()

	game.OnJoinRoom(player)
}

func (this *room) onLeaveRoom(game Game, playerId int64, err error) {
	var p = this.popPlayer(playerId)
	if p == nil {
		return
	}

	if p.Connected() {
		p.Close()
	}

	game.OnLeaveRoom(p, err)
}
