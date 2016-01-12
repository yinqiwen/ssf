package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/glog"
	"github.com/yinqiwen/ssf"
)

type wordCountProcessor struct {
	counts map[string]uint64
	lk     sync.Mutex
}

func (proc *wordCountProcessor) dump() {
	proc.lk.Lock()
	defer proc.lk.Unlock()
	fmt.Printf("Start dump word count results\n")
	for k, c := range proc.counts {
		fmt.Printf("Word:%s Count=%d\n", k, c)
	}
	fmt.Printf("End dump word count results\n")
}

func (proc *wordCountProcessor) parseLine(msg *ssf.RawMessage) {
	ss := strings.Split(string(msg.Data()), " ")
	for _, s := range ss {
		var word Word
		word.Word = &s
		count := int32(1)
		word.Count = &count
		ssf.Emit(&word, ssf.HashCode([]byte(s)))
	}
}

func (proc *wordCountProcessor) count(word *Word) {
	proc.lk.Lock()
	defer proc.lk.Unlock()
	proc.counts[word.GetWord()] += uint64(word.GetCount())
}

func (proc *wordCountProcessor) OnRPC(request proto.Message) proto.Message {
	return nil
}

func (proc *wordCountProcessor) OnMessage(msg proto.Message, hashCode uint64) {
	switch msg.(type) {
	case *ssf.RawMessage:
		proc.parseLine(msg.(*ssf.RawMessage))
	case *Word:
		proc.count(msg.(*Word))
	}
}
func (proc *wordCountProcessor) OnStart() error {
	return nil
}
func (proc *wordCountProcessor) OnStop() error {
	proc.dump()
	return nil
}

func main() {
	home := flag.String("home", "./", "application home dir")
	cluster := flag.String("cluster", "example", "cluster name")
	flag.Parse()
	defer glog.Flush()

	var proc wordCountProcessor
	proc.counts = make(map[string]uint64)
	var config ssf.ProcessorConfig
	config.ClusterName = *cluster
	config.Home = *home
	config.Proc = &proc

	ssf.StartProcessor(&config)
}
