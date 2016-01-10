package ssf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/golang/glog"
	"github.com/yinqiwen/gotoolkit/ots"
)

// type ServerConn struct {
// 	ackWindow uint64 //every 64 events
// }

func processSSFEventConnection(c io.ReadWriteCloser, decodeEvent bool, hander ipcEventHandler) {
	bc := bufio.NewReader(c)
	ignoreMagic := true
	for ssfRunning {
		ev, err := readEvent(bc, ignoreMagic, decodeEvent)
		ignoreMagic = false
		if nil != err {
			if err != io.EOF {
				glog.Errorf("Failed to read event for error:%v from %v", err, c)
			}
			c.Close()
			return
		}
		hander.OnEvent(ev, c)
	}
	c.Close()
}

func processEventConnection(c io.ReadWriteCloser, decodeEvent bool, hander ipcEventHandler) {
	//test connection type
	magicBuf := make([]byte, 4)
	magic, err := readMagicHeader(c, magicBuf)
	if nil != err {
		if err != io.EOF {
			glog.Errorf("Failed to read magic header for error:%v from %v", err, c)
		}
	} else {
		if bytes.Equal(magic, MAGIC_EVENT_HEADER) {
			processSSFEventConnection(c, decodeEvent, hander)
		} else if bytes.EqualFold(magic, MAGIC_OTSC_HEADER) {
			ots.ProcessTroubleShooting(c)
		} else {
			glog.Errorf("Invalid magic header:%s", string(magic))

		}
	}
}

func runServer(l net.Listener) {
	_, isIPCServer := l.(*net.UnixListener)
	ipc := &procIPCEventHandler{}
	for ssfRunning {
		c, _ := l.Accept()
		if nil != c {
			var rwc io.ReadWriteCloser
			rwc = c
			if isIPCServer {
				ic := addIPChannel(c)
				if nil == ic {
					c.Close()
					continue
				}
				rwc = ic
			}
			go processEventConnection(rwc, false, ipc)
		}
	}
	l.Close()
}

func startClusterServer(laddr string) error {
	os.MkdirAll(ssfCfg.ProcHome+"/ipc", 0770)
	ipcAddr := ssfCfg.ProcHome + "/ipc/" + ssfCfg.ClusterName + ".sock"
	if err := trylockFile(ipcAddr); nil != err {
		return fmt.Errorf("IPC file:%s is locked by reason:%v", ipcAddr, err)
	}
	os.Remove(ipcAddr)
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
