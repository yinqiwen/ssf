package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/yinqiwen/ssf"
)

const wordCountEventType = int32(100)

func main() {
	a := flag.String("listen", "127.0.0.1:48100", "listen addr")
	home := flag.String("home", "./", "application home dir")
	cluster := flag.String("cluster", "example/127.0.0.1:48100,127.0.0.1:48101,127.0.0.1:48102", "cluster name&servers")
	flag.Parse()
	defer glog.Flush()

	var cfg ssf.ClusterConfig
	cfg.ListenAddr = *a
	cfg.ProcHome = *home
	cs := strings.Split(*cluster, "/")
	if len(cs) != 2 {
		fmt.Printf("Invalid cluster args.\n")
		flag.PrintDefaults()
		return
	}
	cfg.ClusterName = cs[0]
	cfg.SSFServers = strings.Split(cs[1], ",")
	cfg.Dispatch = map[string][]int32{
		"wc": []int32{1, wordCountEventType},
	}
	ssf.Start(&cfg)
}
