package ssf

import (
	// "encoding/json"
	// "github.com/samuel/go-zookeeper/zk"
	// "time"
	"crypto/md5"
	"encoding/binary"
	//"fmt"
	"github.com/golang/glog"
	//"io/ioutil"
	"net"
	"os"
)

const (
	NODE_ACTIVE  uint8 = 1
	NODE_LOADING uint8 = 2
	NODE_FAULT   uint8 = 3
)

type Node struct {
	Id          int32
	PartitionId int32
	Addr        string
	Status      uint8
}

type Partition struct {
	Id    int32
	Addr  string
	Nodes []int32
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
	allNodes       []Node
	partitions     []Partition
	selfParitionId int32
}

var topo clusterTopo
var ssfCfg ClusterConfig

func getNodeByHash(hashCode uint64) *Node {
	cursor := hashCode & uint64(len(topo.allNodes)-1)
	return &(topo.allNodes[int(cursor)])
}
func getNodeById(id int32) *Node {
	return &(topo.allNodes[int(id)])
}
func isSelfNode(node *Node) bool {
	return node.PartitionId == topo.selfParitionId
}

type ServerData struct {
	Listen string
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
	virtualNodeSize := 128
	for len(ssfCfg.SSFServers) >= virtualNodeSize {
		virtualNodeSize = virtualNodeSize * 2
	}
	host, port, err := net.SplitHostPort(ssfCfg.ListenAddr)
	localAddr := ssfCfg.ListenAddr
	if nil == err && host == "0.0.0.0" {
		h, _ := os.Hostname()
		addrs, _ := net.LookupIP(h)
		for _, addr := range addrs {
			if ipv4 := addr.To4(); ipv4 != nil {
				localAddr = net.JoinHostPort(ipv4.String(), port)
				break
			}
		}
	}
	virtualNodePerPartion := virtualNodeSize / len(ssfCfg.SSFServers)
	topo.allNodes = make([]Node, virtualNodeSize)
	topo.partitions = make([]Partition, len(ssfCfg.SSFServers))
	k := 0
	for i, server := range ssfCfg.SSFServers {
		_, _, err = net.SplitHostPort(server)
		if nil != err {
			glog.Errorf("Invalid server address:%s", server)
			continue
		}
		var partition Partition
		partition.Addr = server
		partition.Id = int32(i)
		partition.Nodes = make([]int32, 0)
		if server == ssfCfg.ListenAddr || server == localAddr {
			topo.selfParitionId = partition.Id
		}
		for j := 0; j < virtualNodePerPartion; j++ {
			var node Node
			node.Id = int32(k)
			node.PartitionId = partition.Id
			node.Addr = server
			node.Status = NODE_ACTIVE
			topo.allNodes[k] = node
			partition.Nodes = append(partition.Nodes, node.Id)
			k++
		}
		if i == len(ssfCfg.SSFServers)-1 {
			for ; k < virtualNodeSize; k++ {
				var node Node
				node.Id = int32(k)
				node.PartitionId = partition.Id
				node.Addr = server
				node.Status = NODE_ACTIVE
				topo.allNodes[k] = node
				partition.Nodes = append(partition.Nodes, node.Id)
			}
		}
		topo.partitions[i] = partition
	}
}

func HashCode(s []byte) uint64 {
	sum := md5.Sum(s)
	a := binary.LittleEndian.Uint64(sum[0:8])
	b := binary.LittleEndian.Uint64(sum[8:16])
	return a ^ b
}

func Start(cfg *ClusterConfig) {
	ssfCfg = *cfg
	if len(cfg.ZookeeperServers) > 0 {
		startClusterServer(ssfCfg.ListenAddr)
	} else if len(cfg.SSFServers) > 0 {
		buildNodeTopoFromConfig()
	} else {
		panic("Invalid config to start ssf.")
	}
	initRoutine()
	if len(ssfCfg.ListenAddr) > 0 {
		err := startClusterServer(ssfCfg.ListenAddr)
		if nil != err {
			panic(err)
		}
	} else {
		panic("No listen addr specified.")
	}
}
