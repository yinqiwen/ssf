package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
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

func (proc *MyProcessor) OnEvent(ev *ssf.Event) error {
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

func readSocket(l net.Listener) {
	process := func(c net.Conn) {
		bc := bufio.NewReader(c)
		scanner := bufio.NewScanner(bc)
		for scanner.Scan() {
			line := scanner.Text()
			rawMsg := ssf.NewRawMessage(line)
			ssf.Emit(rawMsg, ssf.HashCode([]byte(line)))
		}
		c.Close()
	}

	for {
		c, _ := l.Accept()
		if nil != c {
			go process(c)
		}
	}
}

func main() {
	a := flag.String("listen", "127.0.0.1:48100", "listen addr")
	inputAddress := flag.String("inaddr", "127.0.0.1:7788", "input listen addr")
	home := flag.String("home", "", "proc home")
	flag.Parse()
	defer glog.Flush()

	l, err := net.Listen("tcp", *inputAddress)
	if nil != err {
		fmt.Printf("Bind socket failed:%v", err)
		return
	}
	go readSocket(l)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)

	var cfg ssf.ClusterConfig
	cfg.ListenAddr = *a
	cfg.ProcHome = *home
	cfg.SSFServers = []string{"127.0.0.1:48100", "127.0.0.1:48101", "127.0.0.1:48102"}
	var proc MyProcessor
	proc.counts = make(map[string]uint64)
	cfg.Handler = &proc
	//word := Word{}
	ssf.RegisterEvent(WORD_COUNT_EVENT, &Word{})

	go func() {
		_ = <-sc
		proc.dump()
		os.Exit(1)
	}()
	ssf.Start(&cfg)
}
