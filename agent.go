/*
* @Author: yinqiwen
* @Date:   2015-11-24 10:33:22
* @Last Modified by:   wangqiying
* @Last Modified time: 2015-11-30 15:14:19
 */

package ssf

import (
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
)

var zkConn *zk.Conn

//Zookeeper path data
type ServerData struct {
	Addr          string
	Weight        uint32
	ConnectedTime int64
}

type agentEvent struct {
	partitionData string
	nodeData      string
}

var agentEventChannel = make(chan *agentEvent)
var localHostName string
var localHostNamePort string

func watchNodes() {
	path := "/" + ssfCfg.ClusterName + "/topo/nodes"
	data, _, ch, err := zkConn.GetW(path)
	if nil != err {
		glog.Errorf("Failed to retrive nodes from zk:%v", err)
		time.Sleep(1 * time.Second)
	} else {
		agentEventChannel <- &agentEvent{"", string(data)}
		zev := <-ch
		glog.Infof("Receive zk event:%v", zev)
	}
	go watchNodes()
}

func watchPartitions() {
	path := "/" + ssfCfg.ClusterName + "/topo/partitions"
	data, _, ch, err := zkConn.GetW(path)
	if nil != err {
		glog.Errorf("Failed to retrive partitions from zk:%v", err)
		time.Sleep(1 * time.Second)
	} else {
		agentEventChannel <- &agentEvent{string(data), ""}
		zev := <-ch
		glog.Infof("Receive zk event:%v", zev)
	}
	go watchPartitions()
}

func updateClusterParitions(data string) {
	var partitions []Partition
	err := json.Unmarshal([]byte(data), &partitions)
	if nil != err {
		glog.Errorf("Invalid partition json:%s with err:%v", data, err)
	} else {
		buildNodeTopoFromZk(getClusterTopo().allNodes, partitions)
	}
}

func updateClusterNodes(data string) {
	var nodes []Node
	err := json.Unmarshal([]byte(data), &nodes)
	if nil != err {
		glog.Errorf("Invalid nodes json:%s with err:%v", data, err)
	} else {
		buildNodeTopoFromZk(nodes, getClusterTopo().partitions)
	}
}

func buildNodeTopoFromZk(nodes []Node, partitions []Partition) {
	newTopo := new(clusterTopo)
	for _, partition := range partitions {
		if partition.Addr == localHostNamePort {
			newTopo.selfParitionID = partition.Id
			break
		}
	}
	for i := 0; i < len(nodes); i++ {
		for _, partition := range partitions {
			if nodes[i].PartitionID == partition.Id {
				nodes[i].Addr = partition.Addr
				break
			}
		}
	}
	newTopo.allNodes = nodes
	newTopo.partitions = partitions
	glog.Infof("Update route table from zookeeper with self ParitionID:%d", newTopo.selfParitionID)
	glog.Infof("Current cluster partitions is %v.", newTopo.partitions)
	glog.Infof("Current cluster nodes is %v.", newTopo.allNodes)
	saveClusterTopo(newTopo)
}

func createZookeeperPath() error {
	zkPathCreated := false
	var data ServerData
	data.Weight = ssfCfg.Weight
	_, port, err := net.SplitHostPort(ssfCfg.ListenAddr)
	if nil != err {
		return err
	}
	localHostName, err = os.Hostname()
	if nil != err {
		return err
	}
	localHostNamePort = net.JoinHostPort(localHostName, port)
	data.Addr = localHostNamePort
	data.ConnectedTime = time.Now().Unix()
	serverPath := "/" + ssfCfg.ClusterName + "/servers/" + data.Addr
	zkConn.Create("/"+ssfCfg.ClusterName, nil, 0, zk.WorldACL(zk.PermAll))
	zkConn.Create("/"+ssfCfg.ClusterName+"/servers/", nil, 0, zk.WorldACL(zk.PermAll))
	zkData, _ := json.Marshal(&data)
	for !zkPathCreated {
		_, err := zkConn.Create(serverPath, zkData, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if nil != err {
			glog.Errorf("Failed to create zookeeper path:%s with reason:%v", serverPath, err)
			time.Sleep(1 * time.Second)
		} else {
			zkPathCreated = true
		}
	}
	return nil
}

func agentLoop() {
	go watchPartitions()
	go watchNodes()
	for {
		select {
		case aev := <-agentEventChannel:
			if len(aev.nodeData) > 0 {
				updateClusterNodes(aev.nodeData)
			}
			if len(aev.partitionData) > 0 {
				updateClusterParitions(aev.partitionData)
			}
		}
	}
}

func startZkAgent() error {
	c, _, err := zk.Connect(ssfCfg.ZookeeperServers, time.Second*10)
	if err != nil {
		return err
	}
	zkConn = c
	err = createZookeeperPath()
	if err != nil {
		return err
	}
	saveClusterTopo(new(clusterTopo)) //set empty default topo
	go agentLoop()
	return nil
}
