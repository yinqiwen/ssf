package ssf

import (
	"io"
	"net"
	"path/filepath"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/golang/glog"
)

type ipchannel struct {
	name     string
	ech      chan []byte
	closech  chan int
	unixConn net.Conn
}

func (ic *ipchannel) evloop() {
	for ssfRunning {
		if nil == ic.unixConn {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		select {
		case ev := <-ic.ech:
			if len(ev) > 0 {
				ic.unixConn.Write(ev)
			}
		case _ = <-ic.closech:
			ic.unixConn.Close()
			ic.unixConn = nil
		}
	}
}

func (ic *ipchannel) Read(p []byte) (n int, err error) {
	if nil == ic.unixConn {
		return 0, io.EOF
	}
	return ic.unixConn.Read(p)
}

func (ic *ipchannel) Close() error {
	ic.closech <- 1
	return nil
}

func (ic *ipchannel) Write(p []byte) (n int, err error) {
	select {
	case ic.ech <- p:
		return len(p), nil
	default:
		glog.Warningf("Discard msg to processor:%s since it's too slow", ic.name)
		return 0, nil
	}
}

type dispatchTable struct {
	allChannels map[string]*ipchannel
	routeTable  map[int32][]*ipchannel
}

var ssfDispatchTable unsafe.Pointer

func init() {
	dis := new(dispatchTable)
	dis.allChannels = make(map[string]*ipchannel)
	dis.routeTable = make(map[int32][]*ipchannel)
	atomic.StorePointer(&ssfDispatchTable, unsafe.Pointer(dis))
}

func getDispatchTable() *dispatchTable {
	return (*dispatchTable)(atomic.LoadPointer(&ssfDispatchTable))
}
func saveDispatchTable(dis *dispatchTable) {
	atomic.StorePointer(&ssfDispatchTable, unsafe.Pointer(dis))
}

func updateDispatchTable(cfg map[string][]int32) {
	newDispatch := new(dispatchTable)
	newDispatch.allChannels = make(map[string]*ipchannel)
	newDispatch.routeTable = make(map[int32][]*ipchannel)

	oldDispatch := getDispatchTable()
	for proc, types := range cfg {
		if ic, ok := oldDispatch.allChannels[proc]; ok {
			newDispatch.allChannels[proc] = ic
		} else {
			ic := new(ipchannel)
			ic.name = proc
			ic.ech = make(chan []byte, 100000)
			ic.closech = make(chan int)
			newDispatch.allChannels[proc] = ic
			go ic.evloop()
		}

		for _, t := range types {
			newDispatch.routeTable[t] = append(newDispatch.routeTable[t], newDispatch.allChannels[proc])
		}
	}
	saveDispatchTable(newDispatch)
}

func getIPChannelsByType(msgType int32) []*ipchannel {
	ics, ok := getDispatchTable().routeTable[msgType]
	if ok {
		return ics
	}
	return nil
}

func getIPChannelsByName(processor string) *ipchannel {
	ic, ok := getDispatchTable().allChannels[processor]
	if ok {
		return ic
	}
	return nil
}

func addIPChannel(unixConn net.Conn) *ipchannel {
	addr := filepath.Base(unixConn.RemoteAddr().String())
	extension := filepath.Ext(addr)
	proc := addr[0 : len(addr)-len(extension)]
	ic := getIPChannelsByName(proc)
	if nil != ic {
		ic.unixConn = unixConn
	}
	return ic
}

func dispatch(event *Event) error {
	node := getNodeByHash(event.HashCode)
	if nil == node {
		return ErrNoNode
	}
	if isSelfNode(node) {
		ics := getIPChannelsByType(event.MsgType)
		for _, ic := range ics {
			ic.Write(event.Raw)
		}
		return nil
	}
	return emitContent(event.Raw, event.HashCode)
}

type ipcEventHandler struct {
}

func (ipc *ipcEventHandler) OnEvent(event *Event) *Event {
	if event.MsgType == int32(EventType_EVENT_HEARTBEAT) {
		//send heartbeat response back
		var hbres HeartBeat
		res := true
		hbres.Res = &res
		event.Msg = &hbres
		event.Raw = nil
		return event
	}
	dispatch(event)
	return nil
}
