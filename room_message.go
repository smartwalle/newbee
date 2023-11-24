package newbee

func (r *room) onMessage(game Game, playerId int64, data interface{}) {
	var p = r.GetPlayer(playerId)
	if p == nil {
		return
	}
	game.OnMessage(p, data)
}

func (r *room) onDequeue(game Game, data interface{}) {
	game.OnDequeue(data)
}

func (r *room) onJoinRoom(game Game, player Player) error {
	if player == nil {
		return ErrNilPlayer
	}
	r.mu.Lock()

	if _, ok := r.players[player.GetId()]; ok {
		r.mu.Unlock()
		return ErrPlayerExists
	}

	if player.Connected() {
		r.players[player.GetId()] = player

		var sess = player.Session()
		sess.SetId(player.GetId())
		sess.UpdateHandler(r)
	}
	r.mu.Unlock()

	game.OnJoinRoom(player)
	return nil
}

func (r *room) onLeaveRoom(game Game, playerId int64, err error) {
	var p = r.popPlayer(playerId)
	if p == nil {
		return
	}

	if p.Connected() {
		p.Close()
	}

	game.OnLeaveRoom(p, err)
}
