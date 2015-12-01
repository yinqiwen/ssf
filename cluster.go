package ssf

import (
	"crypto/md5"
	"encoding/binary"
	"net"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/golang/glog"
)

const (
	NODE_ACTIVE  uint8 = 1
	NODE_LOADING uint8 = 2
	NODE_FAULT   uint8 = 3
)

type Node struct {
	Id          int32
	PartitionID int32
	Addr        string
	Status      uint8
}

func (node *Node) isActive() bool {
	return node.Status == NODE_ACTIVE
}

type Partition struct {
	Id   int32
	Addr string
	//Nodes []int32
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
	Weight           uint32
}

type clusterTopo struct {
	allNodes       []Node
	partitions     []Partition
	selfParitionID int32
}

//var topo clusterTopo
var ssfCfg ClusterConfig
var ssfTopo unsafe.Pointer

func getClusterTopo() *clusterTopo {
	return (*clusterTopo)(atomic.LoadPointer(&ssfTopo))
}
func saveClusterTopo(topo *clusterTopo) {
	atomic.StorePointer(&ssfTopo, unsafe.Pointer(topo))
}

func getNodeByHash(hashCode uint64) *Node {
	topo := getClusterTopo()
	cursor := hashCode & uint64(len(topo.allNodes)-1)
	return &(getClusterTopo().allNodes[int(cursor)])
}
func getNodeById(id int32) *Node {
	topo := getClusterTopo()
	return &(topo.allNodes[int(id)])
}
func isSelfNode(node *Node) bool {
	topo := getClusterTopo()
	return node.PartitionID == topo.selfParitionID
}
func getClusterNodeSize() int {
	topo := getClusterTopo()
	return len(topo.allNodes)
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
	topo := new(clusterTopo)
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
		//partition.Nodes = make([]int32, 0)
		if server == ssfCfg.ListenAddr || server == localAddr {
			topo.selfParitionID = partition.Id
		}
		for j := 0; j < virtualNodePerPartion; j++ {
			var node Node
			node.Id = int32(k)
			node.PartitionID = partition.Id
			node.Addr = server
			node.Status = NODE_ACTIVE
			topo.allNodes[k] = node
			//partition.Nodes = append(partition.Nodes, node.Id)
			k++
		}
		if i == len(ssfCfg.SSFServers)-1 {
			for ; k < virtualNodeSize; k++ {
				var node Node
				node.Id = int32(k)
				node.PartitionID = partition.Id
				node.Addr = server
				node.Status = NODE_ACTIVE
				topo.allNodes[k] = node
				//partition.Nodes = append(partition.Nodes, node.Id)
			}
		}
		topo.partitions[i] = partition
	}
	saveClusterTopo(topo)
}

//HashCode return the md5 hashcode of raw bytes
func HashCode(s []byte) uint64 {
	sum := md5.Sum(s)
	a := binary.LittleEndian.Uint64(sum[0:8])
	b := binary.LittleEndian.Uint64(sum[8:16])
	return a ^ b
}

//Start launch ssf cluster server
func Start(cfg *ClusterConfig) {
	ssfCfg = *cfg
	if len(cfg.ZookeeperServers) > 0 {
		err := startZkAgent()
		if nil != err {
			panic(err)
		}
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
