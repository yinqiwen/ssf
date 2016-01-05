package ssf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
    "sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
)

var procipc = &ipchannel{}


func init() {
	procipc.name = filepath.Base(os.Args[0])
	procipc.ech = make(chan []byte, 100000)
	procipc.closech = make(chan int)
	go procipc.evloop()
}

//ProcessorStat holds some stat of current processor
type ProcessorStat struct {
	NumOfGORoutines int32
}

//Processor define go version sub processor interface
type Processor interface {
	OnStart() error
	OnStop() error
	OnCommand(cmd string, args []string)(int, string)
	EventHandler
}

var ProcStat ProcessorStat

type ProcessorEventHander struct {
	config *ProcessorConfig
}

func (proc *ProcessorEventHander) OnEvent(event *Event) *Event {
    if atomic.AddUint32(&ProcStat.NumOfGORoutines, 1) > proc.config.MaxGORoutine{
        atomic.AddInt32(&ProcStat.NumOfGORoutines, -1)
        glog.Warningf("Too many[%d] goroutines in processor, discard incoming event.",ProcStat.NumOfGORoutines)
        return nil
    }
    go func(){
    	if event.MsgType == EventType_EVENT_CTRLREQ{
    		ctrl := event.Msg.(*CtrlRequest)
    		  errcode, res := proc.config.Proc.OnCommand(ctrl.GetCmd(), ctrl.GetArgs())

    		}else{
    			proc.config.Proc.OnEvent(event)
    		}
        
        atomic.AddInt32(&ProcStat.NumOfGORoutines, -1)
    }
	return nil
}

//ProcessorConfig is the start option
type ProcessorConfig struct {
	Home         string
	ClusterName  string
	Proc         Processor
	MaxGORoutine int32
}

func connectIPCServer(config *ProcessorConfig) error {
	remote := config.Home + "/ipc/" + config.ClusterName + ".sock"
	local := config.Home + "/ipc/" + filepath.Base(os.Args[0]) + ".sock"
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
	procipc.unixConn = c
	return nil
}

func runProcessor(config *ProcessorConfig) error {
	local := config.Home + "/ipc/" + filepath.Base(os.Args[0]) + ".sock"
	if err := trylockFile(local); nil != err {
		return fmt.Errorf("IPC file:%s is locked by reason:%v", local, err)
	}
	config.Proc.OnStart()
	for {
		err := connectIPCServer(config)
		if nil != err {
			time.Sleep(1 * time.Second)
		}
        hander := &ProcessorEventHander{config}
		processSSFEventConnection(procipc, hander)
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
    if config.MaxGORoutine <= 0{
        config.MaxGORoutine = 100000
    }
	ssfRunning = true
	runProcessor(config)
	return nil
}

//Emit would write msg to SSF framework over IPC unix socket
func Emit(msg proto.Message, hashCode uint64) error {
	if nil == procipc.unixConn {
		return ErrSSFDisconnect
	}
	return WriteEvent(msg, hashCode, procipc)
}
