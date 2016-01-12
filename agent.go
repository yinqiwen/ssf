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
	partitionData []byte
	nodeData      []byte
}

var agentEventChannel = make(chan *agentEvent)
var localHostName string
var localHostNamePort string

func watchNodes() {
	partitionPath := "/" + ssfCfg.ClusterName + "/topo/partitions"
	nodePath := "/" + ssfCfg.ClusterName + "/topo/nodes"
	nodeData, _, ch, err1 := zkConn.GetW(nodePath)
	partitionData, _, err2 := zkConn.Get(partitionPath)
	if nil != err1 || nil != err2 {
		glog.Errorf("Failed to retrive nodes from zk:%v, %v", err1, err2)
		time.Sleep(1 * time.Second)
	} else {
		agentEventChannel <- &agentEvent{partitionData, nodeData}
		zev := <-ch
		glog.Infof("Receive zk event:%v", zev)
	}
	go watchNodes()
}

func updateClusterTopo(nodeData, partitionData []byte) {
	var nodes []Node
	var partitions []Partition
	err := json.Unmarshal(nodeData, &nodes)
	if nil != err {
		glog.Errorf("Invalid nodes json:%s with err:%v", string(nodeData), err)
		return
	}
	err = json.Unmarshal(partitionData, &partitions)
	if nil != err {
		glog.Errorf("Invalid partition json:%s with err:%v", string(partitionData), err)
		return
	}
	buildNodeTopoFromZk(nodes, partitions)
}

func buildNodeTopoFromZk(nodes []Node, partitions []Partition) {
	newTopo := new(clusterTopo)
	for _, partition := range partitions {
		if partition.Addr == localHostNamePort {
			newTopo.selfParitionID = partition.Id
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
	ssfClient.clearInvalidConns()
	ssfClient.checkPartitionConns()
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
	go watchNodes()
	for ssfRunning {
		select {
		case aev := <-agentEventChannel:
			updateClusterTopo(aev.nodeData, aev.partitionData)
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
	for isClusterTopoEmpty() {
		time.Sleep(1 * time.Second)
		glog.Warningf("Cluster topo is empty, wait 1s until agent fetched cluster topo from zk")
	}
	return nil
}
