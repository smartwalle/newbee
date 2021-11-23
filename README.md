## Newbee

Golang 游戏基本框架。

## 安装

```
go get github.com/smartwalle/newbee
```

### 创建房间

```go
var room = newbee.NewRoom(1)
```

### 实现游戏

定义一个结构体，实现 newbee.Game 接口

```go
type Game struct {
}

func (this *Game) GetId() int64 {}
func (this *Game) GetState() newbee.GameState {}
func (this *Game) TickInterval() time.Duration {}
func (this *Game) OnTick()
func (this *Game) OnMessage(player newbee.Player, message interface{})
func (this *Game) OnDequeue(message interface{})
func (this *Game) OnRunInRoom(room newbee.Room)
func (this *Game) OnJoinRoom(player newbee.Player)
func (this *Game) OnLeaveRoom(player newbee.Player)
func (this *Game) OnCloseRoom(room newbee.Room)
func (this *Game) OnPanic(room newbee.Room, err error)
```

### 运行游戏

```go
var game = &Game{}
room.Run(game)
```

### 添加玩家

```go
var player = &Player{} // Player 结构体需要实现 newbee.Player 接口
room.AddPlayer(player)
```