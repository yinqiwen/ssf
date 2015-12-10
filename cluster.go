package ssf

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/golang/glog"
)

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
var ssfRunning bool

func getClusterTopo() *clusterTopo {
	return (*clusterTopo)(atomic.LoadPointer(&ssfTopo))
}
func saveClusterTopo(topo *clusterTopo) {
	atomic.StorePointer(&ssfTopo, unsafe.Pointer(topo))
}

func isClusterTopoEmpty() bool {
	return len(getClusterTopo().allNodes) > 0
}

func getNodeByHash(hashCode uint64) *Node {
	topo := getClusterTopo()
	if len(topo.allNodes) == 0 {
		return nil
	}
	cursor := hashCode & uint64(len(topo.allNodes)-1)
	return &(topo.allNodes[int(cursor)])
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
	ssfRunning = true
	ssfCfg = *cfg
	if nil == cfg.Handler {
		panic("No Handler setting in config.")
	}
	if err := trylockFile(ssfCfg.ProcHome); nil != err {
		panic(fmt.Sprintf("Home:%s is locked by reason:%v", ssfCfg.ProcHome, err))
	}
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
		err := ssfCfg.Handler.OnStart()
		if nil == err {
			err = startClusterServer(ssfCfg.ListenAddr)
		}
		if nil != err {
			panic(err)
		}
	} else {
		panic("No listen addr specified.")
	}
}

//Stop ssf cluster server
func Stop() {
	ssfRunning = false
	ssfCfg.Handler.OnStop()
}
