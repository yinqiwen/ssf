/*
* @Author: yinqiwen
* @Date:   2015-11-24 10:33:22
* @Last Modified by:   wangqiying
* @Last Modified time: 2015-11-25 15:55:01
 */

package ssf

import (
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
)

var zkConn *zk.Conn

func retriveNodes() {

	data, st, ch, err := zkConn.GetW(path)
	if nil != err {
		glog.Errorf("Failed to retrive nodes from zk:%v", err)
	}
	ech := <-ch
}

func retrivePartitions() {
	data, st, ch, err := zkConn.GetW(path)
	if nil != err {
		glog.Errorf("Failed to retrive nodes from zk:%v", err)
	}
	//ech := <-ch
}

func updateClusterParitions(data string) {

}

func updateClusterNodes(data string) {

}

func agentLoop() {

	for {

	}
}

func startZkAgent() error {
	c, _, err := zk.Connect(ssfCfg.ZookeeperServers, time.Second*10)
	if err != nil {
		return err
	}
	zkConn = c
	go agentLoop()
	return nil
}
