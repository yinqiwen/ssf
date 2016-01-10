package ssf

import (
	"fmt"
	"io"
	"net"
	"path/filepath"
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

	node := getNodeByHash(event.GetHashCode())
	if nil == node {
		return ErrNoNode
	}
	if isSelfNode(node) {
		ics := getIPChannelsByType(event.GetMsgType())
		for _, ic := range ics {
			//ic.Write(event.Raw)
			writeEvent(event, ic)
		}
		return nil
	}
	return emitEvent(event)
}

type procIPCEventHandler struct {
}

func (ipc *procIPCEventHandler) OnEvent(event *Event, conn io.ReadWriteCloser) {
	if event.GetType() == EventType_NOTIFY {
		switch event.GetMsgType() {
		case proto.MessageName((*HeartBeat)(nil)):
			var hbres HeartBeat
			res := true
			hbres.Res = &res
			notify(&hbres, 0, conn)
		default:
			dispatch(event)
		}
	} else {
		if event.GetTo() != getProcessorName() {
			dispatch(event)
			return
		}
		event.decode()
		if event.GetType() == EventType_RESPONSE {
			triggerClientSessionRes(event.Msg, event.GetHashCode())
		} else {
			//hanlder event
		}
	}
}

func LocalRPC(processor string, request proto.Message, timeout time.Duration) (proto.Message, error) {
	var channel io.Writer
	if isFrameworkProcessor() {
		ic := getIPChannelsByName(processor)
		if nil == ic || ic.unixConn == nil {
			return nil, fmt.Errorf("No connected processor:%s", processor)
		}
		channel = ic
	} else {
		if procipc.unixConn == nil {
			return nil, ErrSSFDisconnect
		}
		channel = procipc
	}
	return rpc(processor, request, timeout, channel)
}

//LocalCommand send command to given procesor and return response from the processor
func LocalCommand(processor string, cmd string, args []string, timeout time.Duration) (int32, string) {
	var req CtrlRequest
	req.Args = args
	req.Cmd = &cmd
	res, err := LocalRPC(processor, &req, timeout)
	if nil != err {
		return -1, err.Error()
	}
	if ctrlres, ok := res.(*CtrlResponse); !ok {
		return -1, fmt.Sprintf("Invalid type for control response:%T", res)
	} else {
		return ctrlres.GetErrCode(), ctrlres.GetResponse()
	}
	return 0, ""
}
