package ssf

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"github.com/yinqiwen/gotoolkit/gstat"
	"github.com/yinqiwen/gotoolkit/ots"
)

var procipc = &ipchannel{}
var procConfig *ProcessorConfig
var procGORoutinesCounter gstat.CountTrack
var procLatencyTrack gstat.CostTrack
var procQPSTrack gstat.QPSTrack
var isSSFProcessor bool

func init() {
	procipc.name = filepath.Base(os.Args[0])
	procipc.ech = make(chan []byte, 100000)
	procipc.closech = make(chan int)
}

func isFrameworkProcessor() bool {
	return !isSSFProcessor
}

func getProcessorName() string {
	if isFrameworkProcessor() {
		return ssfCfg.ClusterName
	}
	return procConfig.Name
}


type adminEventWriter struct {
	hashCode uint64
}

func (wr *adminEventWriter) Write(p []byte) (n int, err error) {
	var event AdminEvent
	event.Content = proto.String(string(p))
	err = Emit(&event, wr.hashCode)
	if nil != err {
		n = 0
	} else {
		n = len(p)
	}
	return
}

type processorEventHander struct {
}

func (proc *processorEventHander) OnEvent(event *Event, conn io.ReadWriteCloser) {
	procQPSTrack.IncMsgCount(1)
	if procGORoutinesCounter.Add(1) > int64(procConfig.MaxGORoutine) {
		procGORoutinesCounter.Add(-1)
		glog.Warningf("Too many[%d] goroutines in processor, discard incoming event.", procGORoutinesCounter.Get())
		return
	}
	go func() {
		start := time.Now().UnixNano()
		if event.GetType() == EventType_NOTIFY {
			procConfig.Proc.OnMessage(event.Msg, event.GetHashCode())
		} else {
			if event.GetTo() != getProcessorName() {
				fmt.Errorf("Recv msg %T to %s, but current processor is %s", event.Msg, event.GetTo(), getProcessorName())
				return
			}
			if event.GetType() == EventType_RESPONSE {
				triggerClientSessionRes(event.Msg, event.GetHashCode())
			} else {
				var res proto.Message
				if admin, ok := event.Msg.(*AdminRequest); ok {
					err := ots.Handle(admin.GetLine(), &adminEventWriter{event.GetHashCode()})
					res = &AdminResponse{}
					if err == io.EOF {
						res.(*AdminResponse).Close = proto.Bool(true)
					}
				} else {
					res = procConfig.Proc.OnRPC(event.Msg)
				}
				response(res, event, procipc)
			}
		}
		end := time.Now().UnixNano()
		cost := (end - start) / 1000000 //millsecs
		procLatencyTrack.AddCost(cost)
		procGORoutinesCounter.Add(-1)
	}()
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for s := range c {
			// Stop when receive signal Interrupt
			if s == os.Interrupt {
				config.Proc.OnStop()
				os.Exit(0)
			}
		}
	}()

	for {
		err := connectIPCServer(config)
		if nil != err {
			time.Sleep(1 * time.Second)
			continue
		}
		hander := &processorEventHander{}
		processEventConnection(procipc, true, hander)
		//procipc.Close()
	}
}

func startProcessor(config *ProcessorConfig) error {
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
	isSSFProcessor = true
	procConfig = config
	runProcessor(config)
	return nil
}
