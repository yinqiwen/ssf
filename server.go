package ssf

import (
	"bufio"
	"github.com/golang/glog"
	"net"
)

type ServerConn struct {
	ackWindow uint64 //every 64 events

}

func startClusterServer(laddr string) error {
	l, err := net.Listen("tcp", laddr)
	if nil != err {
		return err
	}

	process := func(c net.Conn) {
		bc := bufio.NewReader(c)
		for {
			ev, err := readEvent(bc)
			if nil != err {
				return
			} else {
				if ev.MsgType == int32(EventType_EVENT_HEARTBEAT) {
					//send heartbeat response back
					var hbres HeartBeat
					res := true
					hbres.Res = &res
					ev.Msg = &hbres
					writeEvent(ev, c)
				} else {
					ssfCfg.Handler.OnEvent(ev)
				}
			}
		}
	}

	for {
		c, _ := l.Accept()
		if nil != c {
			go process(c)
		}
	}
}
