package ssf

import (
	"io"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
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
			glog.Infof("Processor:%s disconnect.", ic.name)
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
	if nil == ic.unixConn || !ssfRunning {
		if nil != ic.unixConn {
			ic.unixConn.Close()
			ic.unixConn = nil
		}
		return nil
	}
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
	routeTable  map[string][]*ipchannel
}

var ssfDispatchTable unsafe.Pointer

func init() {
	dis := new(dispatchTable)
	dis.allChannels = make(map[string]*ipchannel)
	dis.routeTable = make(map[string][]*ipchannel)
	atomic.StorePointer(&ssfDispatchTable, unsafe.Pointer(dis))
}

func getDispatchTable() *dispatchTable {
	return (*dispatchTable)(atomic.LoadPointer(&ssfDispatchTable))
}
func saveDispatchTable(dis *dispatchTable) {
	atomic.StorePointer(&ssfDispatchTable, unsafe.Pointer(dis))
}

func updateDispatchTable(cfg map[string][]string) {
	newDispatch := new(dispatchTable)
	newDispatch.allChannels = make(map[string]*ipchannel)
	newDispatch.routeTable = make(map[string][]*ipchannel)

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

func getIPChannelsByType(msgType string) []*ipchannel {
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
		glog.Infof("Processor:%s connected.", proc)
		ic.unixConn = unixConn
	} else {
		glog.Warningf("No processor:%s defined in IPC disptach table.", proc)
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
			//ic.Write(event.Raw)
			writeEvent(event, ic)
		}
		return nil
	}
	return emitEvent(event)
}

var commandSessions = make(map[uint64]chan *CtrlResponse)
var commandSessionIDSeed = uint64(0)
var commondSessionsLock sync.Mutex

func addCommandSession() (uint64, chan *CtrlResponse) {
	commondSessionsLock.Lock()
	defer commondSessionsLock.Unlock()
	id := commandSessionIDSeed
	commandSessionIDSeed++
	ch := make(chan *CtrlResponse)
	commandSessions[id] = ch
	return id, ch
}

func triggerCommandSessionRes(res *CtrlResponse, hashCode uint64) {
	commondSessionsLock.Lock()
	defer commondSessionsLock.Unlock()
	ch, ok := commandSessions[hashCode]
	if !ok {
		return
	}
	delete(commandSessions, hashCode)
	ch <- res
}

type ipcEventHandler struct {
}

func (ipc *ipcEventHandler) OnEvent(event *Event) *Event {
	switch event.MsgType {
	case proto.MessageName((*HeartBeat)(nil)):
		var hbres HeartBeat
		res := true
		hbres.Res = &res
		hbevent := &Event{}
		hbevent.Msg = &hbres
		return hbevent
	case proto.MessageName((*CtrlResponse)(nil)):
		event.Decode()
		triggerCommandSessionRes(event.Msg.(*CtrlResponse), event.Sequence)
	default:
		dispatch(event)
	}
	return nil
}

//LocalCommand send command to given procesor and return response from the processor
func LocalCommand(processor string, cmd string, args []string, timeout time.Duration) (int32, string) {
	ic := getIPChannelsByName(processor)
	if nil == ic || ic.unixConn == nil {
		return -1, "No connected processor."
	}
	timeoutTicker := time.NewTicker(timeout).C
	var req CtrlRequest
	req.Args = args
	req.Cmd = &cmd
	id, rch := addCommandSession()
	defer close(rch)
	WriteEvent(&req, 0, id, ic)
	select {
	case res := <-rch:
		return res.GetErrCode(), res.GetResponse()
	case <-timeoutTicker:
		return -1, ErrCommandTimeout.Error()
	}
	return 0, ""
}
