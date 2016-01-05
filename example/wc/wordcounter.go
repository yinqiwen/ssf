package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/yinqiwen/ssf"
)

const WORD_COUNT_EVENT = 100

type MyProcessor struct {
	counts map[string]uint64
}

func (proc *MyProcessor) dump() {
	fmt.Printf("Start dump word count results\n")
	for k, c := range proc.counts {
		fmt.Printf("Word:%s Count=%d\n", k, c)
	}
	fmt.Printf("\n")
}

func (proc *MyProcessor) parseLine(msg *ssf.RawMessage) {
	ss := strings.Split(string(msg.Data()), " ")
	for _, s := range ss {
		var word Word
		word.Word = &s
		count := int32(1)
		word.Count = &count
		ssf.Emit(&word, ssf.HashCode([]byte(s)))
	}
}

func (proc *MyProcessor) count(word *Word) {
	proc.counts[word.GetWord()] += uint64(word.GetCount())
}

func (proc *MyProcessor) OnEvent(ev *ssf.Event) *Event {
	if ev.MsgType == int32(ssf.EventType_EVENT_RAW) {
		proc.parseLine(ev.Msg.(*ssf.RawMessage))
	} else if ev.MsgType == int32(WORD_COUNT_EVENT) {
		proc.count(ev.Msg.(*Word))
	}
	return nil
}
func (proc *MyProcessor) OnStart() error {
	return nil
}
func (proc *MyProcessor) OnStop() error {
	return nil
}

// func readSocket(l net.Listener) {
// 	process := func(c net.Conn) {
// 		bc := bufio.NewReader(c)
// 		scanner := bufio.NewScanner(bc)
// 		for scanner.Scan() {
// 			line := scanner.Text()
// 			rawMsg := ssf.NewRawMessage(line)
// 			ssf.Emit(rawMsg, ssf.HashCode([]byte(line)))
// 		}
// 		c.Close()
// 	}

// 	for {
// 		c, _ := l.Accept()
// 		if nil != c {
// 			go process(c)
// 		}
// 	}
// }

func main() {
	flag.Parse()
	defer glog.Flush()
	//word := Word{}
	ssf.RegisterEvent(WORD_COUNT_EVENT, &Word{})

	var config ssf.ProcessorConfig
	config.ClusterName = ""
	config.Proc = &MyProcessor{}
	ssf.StartProcessor(&config)
}
