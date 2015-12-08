package ssf

import (
	"bufio"
	"bytes"
	"io"
	"net"

	"github.com/golang/glog"
	"github.com/yinqiwen/gotoolkit/ots"
)

const (
	SSF_EVENT_CONN = 1
	OTS_DEBUG_CONN = 2
)

type ServerConn struct {
	ackWindow uint64 //every 64 events
}

func processSSFEventConnection(c net.Conn) {
	bc := bufio.NewReader(c)
	ignoreMagic := true
	for {
		ev, err := readEvent(bc, ignoreMagic)
		ignoreMagic = false
		if nil != err {
			if err != io.EOF {
				glog.Errorf("Failed to read event for error:%v from %v", err, c.RemoteAddr())
			}
			c.Close()
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

func startClusterServer(laddr string) error {
	l, err := net.Listen("tcp", laddr)
	if nil != err {
		return err
	}

	process := func(c net.Conn) {
		//test connection type
		magicBuf := make([]byte, 4)
		magic, err := readMagicHeader(c, magicBuf)
		if nil != err {
			if err != io.EOF {
				glog.Errorf("Failed to read magic header for error:%v from %v", err, c.RemoteAddr())
			}
			c.Close()
			return
		}
		if bytes.Equal(magic, MAGIC_EVENT_HEADER) {
			processSSFEventConnection(c)
		} else if bytes.EqualFold(magic, MAGIC_OTSC_HEADER) {
			ots.ProcessTroubleShooting(c)
		} else {
			glog.Errorf("Invalid magic header:%s", string(magic))
			c.Close()
		}
	}

	for {
		c, _ := l.Accept()
		if nil != c {
			go process(c)
		}
	}
}
