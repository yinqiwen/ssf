package ssf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"github.com/yinqiwen/gotoolkit/gstat"
)

var procipc = &ipchannel{}
var procGORoutinesCounter gstat.CountTrack
var procLatencyTrack gstat.CostTrack
var procQPSTrack gstat.QPSTrack

func init() {
	procipc.name = filepath.Base(os.Args[0])
	procipc.ech = make(chan []byte, 100000)
	procipc.closech = make(chan int)

}

//Processor define go version sub processor interface
type Processor interface {
	OnStart() error
	OnStop() error
	OnCommand(cmd string, args []string) (int32, string)
	EventHandler
}

type processorEventHander struct {
	config *ProcessorConfig
}

func (proc *processorEventHander) OnEvent(event *Event) *Event {
	procQPSTrack.IncMsgCount(1)
	if procGORoutinesCounter.Add(1) > int64(proc.config.MaxGORoutine) {
		procGORoutinesCounter.Add(-1)
		glog.Warningf("Too many[%d] goroutines in processor, discard incoming event.", procGORoutinesCounter.Get())
		return nil
	}
	go func() {
		switch event.Msg.(type) {
		case *CtrlRequest:
			req := event.Msg.(*CtrlRequest)
			errcode, reason := proc.config.Proc.OnCommand(req.GetCmd(), req.GetArgs())
			res := &CtrlResponse{}
			res.Response = &reason
			res.ErrCode = &errcode
			WriteEvent(res, event.HashCode, event.Sequence, procipc)
		default:
			start := time.Now().UnixNano()
			proc.config.Proc.OnEvent(event)
			end := time.Now().UnixNano()
			cost := (end - start) / 1000000 //millsecs
			procLatencyTrack.AddCost(cost)
		}
		procGORoutinesCounter.Add(-1)
	}()
	return nil
}

//ProcessorConfig is the start option
type ProcessorConfig struct {
	Home         string
	ClusterName  string
	Proc         Processor
	MaxGORoutine int32
	Name         string
}

func connectIPCServer(config *ProcessorConfig) error {
	os.MkdirAll(config.Home+"/ipc", 0770)
	remote := config.Home + "/ipc/" + config.ClusterName + ".sock"
	local := config.Home + "/ipc/" + config.Name + ".sock"
	//Unix client would connect failed if the unix socket file exist, so remove it first
	os.Remove(local)

	var dialer net.Dialer
	//Bind local unix socket
	dialer.LocalAddr = &net.UnixAddr{local, "unix"}
	c, err := dialer.Dial("unix", remote)
	if nil != err {
		glog.Warningf("Failed to connect ssf framework for reason:%v", err)
		return err
	}
	glog.Infof("SSF framwork IPC server connected.")
	procipc.unixConn = c
	return nil
}

func runProcessor(config *ProcessorConfig) error {
	local := config.Home + "/ipc/" + config.Name + ".sock"
	if err := trylockFile(local); nil != err {
		return fmt.Errorf("IPC file:%s is locked by reason:%v", local, err)
	}
	config.Proc.OnStart()
	go procipc.evloop()
	for {
		err := connectIPCServer(config)
		if nil != err {
			time.Sleep(1 * time.Second)
			continue
		}
		hander := &processorEventHander{config}
		processEventConnection(procipc, true, hander)
		//procipc.Close()
	}
}

//StartProcessor init & launch go processor
func StartProcessor(config *ProcessorConfig) error {
	if len(config.ClusterName) == 0 {
		return ErrNoClusterName
	}
	if len(config.Home) == 0 {
		config.Home = "./"
	}
	if len(config.Name) == 0 {
		config.Name = filepath.Base(os.Args[0])
	}
	if config.MaxGORoutine <= 0 {
		config.MaxGORoutine = 100000
	}
	gstat.AddStatTrack("NumOfGORoutinesOf"+config.Name, &procGORoutinesCounter)
	gstat.AddStatTrack("LatencyOf"+config.Name, &procLatencyTrack)
	gstat.AddStatTrack("QPSOf"+config.Name, &procQPSTrack)
	ssfRunning = true
	runProcessor(config)
	return nil
}

//Emit would write msg to SSF framework over IPC unix socket
func Emit(msg proto.Message, hashCode uint64) error {
	if nil == procipc.unixConn {
		return ErrSSFDisconnect
	}
	return WriteEvent(msg, hashCode, 0, procipc)
}
