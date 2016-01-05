package ssf

import (
	"bufio"
	"bytes"
	"io"
	"net"

	"github.com/golang/glog"
	"github.com/yinqiwen/gotoolkit/ots"
)

// type ServerConn struct {
// 	ackWindow uint64 //every 64 events
// }

func processSSFEventConnection(c io.ReadWriteCloser, hander EventHandler) {
	bc := bufio.NewReader(c)
	ignoreMagic := true
	for ssfRunning {
		ev, err := readEvent(bc, ignoreMagic, true)
		ignoreMagic = false
		if nil != err {
			if err != io.EOF {
				glog.Errorf("Failed to read event for error:%v from %v", err, c)
			}
			c.Close()
			return
		}
		ev = hander.OnEvent(ev)
		if nil != ev {
			writeEvent(ev, c)
		}
	}
}

func processEventConnection(c io.ReadWriteCloser, hander EventHandler) {
	//test connection type
	magicBuf := make([]byte, 4)
	magic, err := readMagicHeader(c, magicBuf)
	if nil != err {
		if err != io.EOF {
			glog.Errorf("Failed to read magic header for error:%v from %v", err, c)
		}
	} else {
		if bytes.Equal(magic, MAGIC_EVENT_HEADER) {
			processSSFEventConnection(c, hander)
		} else if bytes.EqualFold(magic, MAGIC_OTSC_HEADER) {
			ots.ProcessTroubleShooting(c)
		} else {
			glog.Errorf("Invalid magic header:%s", string(magic))

		}
	}
	c.Close()
}

func runServer(l net.Listener) {
	isIPCServer := false
	ipc := &ipcEventHandler{}
	for ssfRunning {
		c, _ := l.Accept()
		if nil != c {
			var rwc io.ReadWriteCloser
			rwc = c
			if isIPCServer {
				rwc = addIPChannel(c)
			}
			go processEventConnection(rwc, ipc)
		}
	}
	l.Close()
}

func startClusterServer(laddr string) error {
	ipcAddr := ssfCfg.ProcHome + "/ipc/" + ssfCfg.ClusterName + ".sock"
	l, err := net.Listen("unix", ipcAddr)
	if nil != err {
		return err
	}
	go runServer(l)
	l, err = net.Listen("tcp", laddr)
	if nil != err {
		return err
	}
	runServer(l)
	return nil
}
