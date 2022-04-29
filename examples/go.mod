module github.com/smartwalle/newbee/examples

require (
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/marten-seemann/qtls v0.10.0 // indirect
	github.com/smartwalle/net4go v0.0.49
	//github.com/smartwalle/net4go/quic v0.0.5
	github.com/smartwalle/net4go/ws v0.0.20
	github.com/smartwalle/newbee v0.0.35
	go.opencensus.io v0.22.2 // indirect
	github.com/smartwalle/queue v0.0.1
)

replace github.com/smartwalle/newbee => ../

go 1.18
