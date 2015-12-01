package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/yinqiwen/ssf"
)

var chash *ssf.Consistent

//var watchEventCh = make()

func watchLoop(zconn *zk.Conn) {
	//zconn.ChildrenW
	for {

	}
}

func main() {
	zks := flag.String("zk", "127.0.0.1:2181,127.0.0.1:2182", "zookeeper servers")
	root := flag.String("root", "mycluster", "zookeeper root path")

	zkServers := strings.Split(*zks, ",")
	c, _, err := zk.Connect(zkServers, time.Second*10)
	if nil != err {
		fmt.Printf("Connect %v failed with reason:%v", err)
		return
	}
	lock := zk.NewLock(c, "/"+*root, nil)
	err = lock.Lock()
	if nil != err {
		fmt.Printf("Lock failed with reason:%v", err)
		return
	}
	defer lock.Unlock()
	//consistentHash.

}
