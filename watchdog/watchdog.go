package main

import (
	"encoding/json"
	"flag"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/yinqiwen/ssf"
)

var zkAcls []zk.ACL
var zconn *zk.Conn
var consitent *ssf.Consistent
var rootZkPath string
var serversZkPath string
var topoNodesPath string
var topoPartitionsPath string

var nodeUptime = uint64(10)
var nodeDowntime = uint64(30)

type cacheServer struct {
	addr     string
	uptime   int64
	downtime int64
	touch    bool
}

var cacheServersLock sync.Mutex
var cacheServers = make(map[string]*cacheServer)

func currentConnectedServes() []string {
	cacheServersLock.Lock()
	defer cacheServersLock.Unlock()
	var ss []string
	for _, server := range cacheServers {
		now := time.Now().Unix()
		if server.downtime > 0 {
			glog.Warningf("Server:%s is down  %ds ago.", server.addr, now-server.downtime)
			if now-server.downtime > int64(nodeDowntime) {
				delete(cacheServers, server.addr)
				continue
			}
		}
		if now-server.uptime > int64(nodeUptime) {
			ss = append(ss, server.addr)
		} else {
			glog.Infof("Server:%s is up %ds ago.", server.addr, now-server.uptime)
		}
	}
	return ss
}
func currentClusterServes() []string {
	var ss []string
	for _, server := range consitent.Servers() {
		ss = append(ss, server.Addr)
	}
	return ss
}

func diffStrings(a, b []string) []string {
	sort.Strings(a)
	sort.Strings(b)
	i := 0
	j := 0
	var res []string
	for i < len(a) {
		if j == len(b) {
			res = append(res, a[i:]...)
			return res
		}
		if a[i] < b[j] {
			res = append(res, a[i])
			i++
		} else {
			if b[j] >= a[i] {
				i++
			}
			j++
		}
	}
	return res
}

func watchChildren() {
	dirs, _, ch, err := zconn.ChildrenW(serversZkPath)
	if nil != err {
		glog.Errorf("Failed to watch servers from zk:%v", err)
		time.Sleep(1 * time.Second)
	} else {
		var ss []ssf.ServerData
		for _, dir := range dirs {
			data, _, err := zconn.Get(serversZkPath + "/" + dir)
			if nil == err {
				var server ssf.ServerData
				err = json.Unmarshal(data, &server)
				if nil == err {
					ss = append(ss, server)
				}
			}
			if nil != err {
				glog.Errorf("Error occured:%v", err)
			}
		}
		cacheServersLock.Lock()
		for _, s := range ss {
			server, ok := cacheServers[s.Addr]
			if !ok {
				server = &cacheServer{}
				cacheServers[s.Addr] = server
			}
			server.addr = s.Addr
			server.downtime = 0
			server.uptime = s.ConnectedTime
			server.touch = true
		}
		for _, s := range cacheServers {
			if !s.touch && s.downtime == 0 {
				s.downtime = time.Now().Unix()
			}
			s.touch = false
		}
		cacheServersLock.Unlock()
		zev := <-ch
		glog.Infof("Receive zk event:%v", zev)
	}
	go watchChildren()
}

func watchdogProcess() {
	go watchChildren()
	//1. Build initial cluster topo
	var partitions []ssf.Partition
	var nodes []ssf.Node
	for {
		var data []byte
		var err error
		data, _, err = zconn.Get(topoPartitionsPath)
		if nil == err && len(data) > 0 {
			err = json.Unmarshal(data, &partitions)
			if nil == err {
				data, _, err = zconn.Get(topoNodesPath)
				if nil == err {
					err = json.Unmarshal(data, &nodes)
				}
			}
		}
		if nil != err {
			glog.Errorf("Failed to watch servers from zk:%v", err)
			time.Sleep(1 * time.Second)
			continue
		} else {
			break
		}
	}
	glog.Infof("Retrive old partitions:%v", partitions)
	glog.Infof("Retrive old nodes:%v", nodes)
	for _, node := range nodes {
		for i := 0; i < len(partitions); i++ {
			if node.PartitionID == partitions[i].Id {
				partitions[i].Nodes = append(partitions[i].Nodes, node.Id)
				break
			}
		}
	}
	for _, partition := range partitions {
		consitent.Set(partition)
	}
	consitent.Update()

	//2. check connected servers every 10s
	ticker := time.NewTicker(time.Second * 10).C
	for {
		select {
		case <-ticker:
			connected := currentConnectedServes()
			active := currentClusterServes()
			removed := diffStrings(active, connected)
			added := diffStrings(connected, active)
			newConsistent := ssf.NewConsistentCopy(consitent)
			if len(removed) > 0 || len(added) > 0 {
				for _, s := range removed {
					newConsistent.Remove(s)
				}
				for _, s := range added {
					newConsistent.Add(s)
				}
				newConsistent.Update()
				topoPartitions := newConsistent.Servers()
				var topoNodes []ssf.Node
				for i := 0; i < len(topoPartitions); i++ {
					for j := 0; j < len(topoPartitions[i].Nodes); j++ {
						node := ssf.Node{topoPartitions[i].Nodes[j], topoPartitions[i].Id, "", ssf.NODE_ACTIVE}
						topoNodes = append(topoNodes, node)
					}
					topoPartitions[i].Nodes = nil
				}
				partitionData, _ := json.Marshal(topoPartitions)
				nodeData, _ := json.Marshal(topoNodes)
				setParitions := &zk.SetDataRequest{topoPartitionsPath, partitionData, -1}
				setNodes := &zk.SetDataRequest{topoNodesPath, nodeData, -1}
				_, err := zconn.Multi(setParitions, setNodes)
				if nil != err {
					glog.Errorf("Failed to update topo with reason:%v", err)
				} else {
					glog.Infof("Update topo partitions to %s", string(partitionData))
					glog.Infof("Update topo nodes to %s", string(nodeData))
					consitent = newConsistent
				}
			}
		}
	}
}

func main() {
	zks := flag.String("zk", "127.0.0.1:2181,127.0.0.1:2182", "zookeeper servers")
	root := flag.String("root", "mycluster", "zookeeper root path")
	numOfNodes := flag.Uint("nsize", 128, "cluster virtual node size, default 128")
	flag.Uint64Var(&nodeDowntime, "downtime", 30, "erase cluster node after downtime(secs)")
	flag.Uint64Var(&nodeUptime, "uptime", 10, "add cluster node after uptime(secs)")
	flag.Parse()
	defer glog.Flush()

	zkAcls = zk.WorldACL(zk.PermAll)

	zkServers := strings.Split(*zks, ",")
	c, _, err := zk.Connect(zkServers, time.Second*10)
	if nil != err {
		glog.Errorf("Connect %v failed with reason:%v", zkServers, err)
		return
	}
	rootZkPath = "/" + *root
	serversZkPath = rootZkPath + "/servers"
	topoNodesPath = rootZkPath + "/topo/nodes"
	topoPartitionsPath = rootZkPath + "/topo/partitions"
	c.Create(rootZkPath, nil, 0, zkAcls)

	lock := zk.NewLock(c, rootZkPath+"/lock", zkAcls)
	err = lock.Lock()
	if nil != err {
		glog.Errorf("Lock failed with reason:%v", err)
		return
	}
	defer lock.Unlock()
	c.Create(serversZkPath, nil, 0, zkAcls)
	c.Create(rootZkPath+"/topo", nil, 0, zkAcls)
	c.Create(topoNodesPath, nil, 0, zkAcls)
	c.Create(topoPartitionsPath, nil, 0, zkAcls)
	zconn = c
	consitent = ssf.NewConsistent(int(*numOfNodes))
	watchdogProcess()
}
