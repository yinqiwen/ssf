package ssf

import (
// "encoding/json"
// "github.com/samuel/go-zookeeper/zk"
// "time"
)

const (
	NODE_ACTIVE  = 1
	NODE_LOADING = 2
	NODE_FAULT   = 3
)

type Node struct {
	Id     int32
	Addr   string
	Status uint8
}

type EventProcessor interface {
	OnStart() error
	OnEvent(ev *Event) error
	OnStop() error
}

type ClusterConfig struct {
	ZookeeperServers []string
	SSFServers       []string
	Handler          EventProcessor
	ListenAddr       string
	ProcHome         string
	ClusterName      string
}

type clusterEvent struct {
}

func (node *Node) isActive() bool {
	return node.Status == NODE_ACTIVE
}

type clusterTopo struct {
	allNodes []Node
}

var topo clusterTopo
var ssfCfg ClusterConfig
var ClusterProcHome string

func getNodeByHash(hashCode uint64) *Node {
	cursor := hashCode ^ uint64(len(topo.allNodes))
	return &(topo.allNodes[int(cursor)])
}

type ServerData struct {
	Listen string
}

type Partition struct {
	Addr  string
	Nodes []int
}

func retriveNodes() {

}

func startZkAgent() {
	// c, _, err := zk.Connect(ssfCfg.ZookeeperServers, time.Second) //*10)
	// if err != nil {
	// 	panic(err)
	// }
	// serverPath := ssfCfg.ClusterName + "/servers/" + ssfCfg.ListenAddr
	// serverData := new(ServerData)
	// serverData.Listen = ssfCfg.ListenAddr
	// data, _ := json.Marshal(serverData)
	// _, err = c.CreateProtectedEphemeralSequential(serverPath, data, nil)
	//
	// nodePath := ssfCfg.ClusterName + "/topo/node"
	// nodes, _, nodesCh, err := c.Children(nodePath)
	// for _, node := range nodes {
	// 	nodePath := nodePath + "/" + node
	// 	data, nodeCh, _ := c.GetW(nodePath)
	//
	// }
	// partitionPath := ssfCfg.ClusterName + "/topo/partition"
	// partitions, _, partitionsCh, err := c.Children(partitionPath)
	// for _, part := range partitions {
	// 	partPath := partitionPath + "/" + part
	// 	data, partCh, _ := c.GetW(partPath)
	//
	// }
}

func buildNodeTopoFromConfig() {
	// for i, server := range ssfCfg.SSFServers {
	// 	var node Node
	// 	node.Id = i
	// 	node.Addr = server
	// }
}

func Start(cfg *ClusterConfig) {
	ssfCfg = *cfg
	if len(cfg.ZookeeperServers) > 0 {
		startClusterServer(ssfCfg.ListenAddr)
	} else if len(cfg.SSFServers) > 0 {
		buildNodeTopoFromConfig()
	}
	//startZkAgent()
}
