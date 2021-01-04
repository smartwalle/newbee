module github.com/smartwalle/newbee/cmd

go 1.12

require (
	github.com/gorilla/websocket v1.4.2
	github.com/smartwalle/net4go v0.0.39
	github.com/smartwalle/net4go/quic v0.0.4
	github.com/smartwalle/net4go/ws v0.0.10
	github.com/smartwalle/newbee v0.0.35
)

replace github.com/smartwalle/newbee => ../
