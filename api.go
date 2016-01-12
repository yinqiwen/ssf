package ssf

import (
	"io"
	"time"

	"github.com/golang/protobuf/proto"
)

//Notify write proto message with given hashcode to writer
func Notify(msg proto.Message, hashCode uint64, writer io.Writer) error {
	return notify(msg, hashCode, writer)
}

//API for SSF framework processor

//ClusterConfig :SSF cluster launch option
type ClusterConfig struct {
	ZookeeperServers []string
	SSFServers       []string
	//Handler          EventProcessor
	ListenAddr  string
	ProcHome    string
	ClusterName string
	Weight      uint32
	Dispatch    map[string][]string
}

//UpdateConfig update current config on the fly
func UpdateConfig(cfg *ClusterConfig) {
	//Currently, only update processor dispatch config
	updateDispatchTable(cfg.Dispatch)
}

//Start launch ssf cluster server
func Start(cfg *ClusterConfig) {
	start(cfg)
}

//LocalRPC would RPC specified processor with given proto request & wait for response
func LocalRPC(processor string, request proto.Message, attach interface{}, timeout time.Duration) (proto.Message, error) {
	var channel io.Writer
	if isFrameworkProcessor() {
		ic := getIPChannelsByName(processor)
		if nil == ic || ic.unixConn == nil {
			return nil, ErrProcessorDisconnect
		}
		channel = ic
	} else {
		if procipc.unixConn == nil {
			return nil, ErrProcessorDisconnect
		}
		channel = procipc
	}
	return rpc(processor, request, timeout, attach, channel)
}

//Stop ssf cluster server
func Stop() {
	stop()
}

//API for golang processor

//Processor define  sub processor interface
type Processor interface {
	OnStart() error
	OnStop() error
	OnRPC(request proto.Message) proto.Message
	OnMessage(msg proto.Message, hashCode uint64)
}

//ProcessorConfig is the start option
type ProcessorConfig struct {
	Home         string
	ClusterName  string
	Proc         Processor
	MaxGORoutine int32
	Name         string
}

//StartProcessor init & launch go processor
func StartProcessor(config *ProcessorConfig) error {
	return startProcessor(config)
}

//Emit would write msg to SSF framework over IPC unix socket
func Emit(msg proto.Message, hashCode uint64) error {
	if nil == procipc.unixConn {
		return ErrProcessorDisconnect
	}
	return notify(msg, hashCode, procipc)
}

//HashCode return the md5 hashcode of raw bytes
func HashCode(s []byte) uint64 {
	return stringHashCode(s)
}
