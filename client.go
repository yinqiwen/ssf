package ssf

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
)

var ssfClient clusterClient

type NodeIOEvent struct {
	wal   *WAL
	event []byte
}

func encodeNodeEvent(msg proto.Message, hashCode uint64) []byte {
	var buf bytes.Buffer
	notify(msg, hashCode, &buf)
	// ev := new(NodeIOEvent)
	// ev.event = buf.Bytes()
	return buf.Bytes()
}

func newEmptyNodeEvent() *NodeIOEvent {
	ev := new(NodeIOEvent)
	ev.event = make([]byte, 0)
	return ev
}

// nodeConn wraps a connection, usually a persistent one
type rawTCPConn struct {
	addr       string
	conn       net.Conn
	writech    chan *NodeIOEvent
	closech    chan int
	closed     bool
	hbPongTime int64 //the time recv heatbeat pong
}

func (nc *rawTCPConn) close() {
	if !nc.closed {
		nc.closed = true
		nc.closech <- 1
	}
}

func (nc *rawTCPConn) logUnsentEvents() {
	for {
		select {
		case ev := <-nc.writech:
			ev.wal.Write(ev.event)
		default:
			return
		}
	}
}

func (nc *rawTCPConn) evloop() {
	for {
		select {
		case ev := <-nc.writech:
			if nil != ev {
				if nil != nc.conn && ev.wal.empty() && len(ev.event) > 0 {
					n, err := nc.conn.Write(ev.event)
					if nil == err {
						continue
					}
					if n != len(ev.event) {
						glog.Warningf("Failed to write %d bytes(writed %d) to %s for reason:%v", len(ev.event), n, nc.addr, err)
					}
					nc.close()
				}
				if len(ev.event) > 0 {
					ev.wal.Write(ev.event)
				}
				if nil != nc.conn && !nc.closed && !ev.wal.empty() {
					//go ev.wal.replay(nc)
					err := ev.wal.replay(nc.conn)
					if nil != err {
						nc.close()
					}
				}
			}
		case _ = <-nc.closech:
			if nil != nc.conn {
				nc.conn.Close()
			}
			nc.conn = nil
			ssfClient.eraseConn(nc)
			nc.logUnsentEvents()
			return
		}
	}
}

func (nc *rawTCPConn) readloop() {
	var bc *bufio.Reader
	reconnectAfter := 1 * time.Second
	for !nc.closed {
		if nil == nc.conn {
			var err error
			nc.conn, err = net.DialTimeout("tcp", nc.addr, 5*time.Second)
			if nil != err {
				glog.Warningf("Failed to connect %s for reason:%v", nc.addr, err)
				time.Sleep(reconnectAfter)
				continue
			}
			bc = bufio.NewReader(nc.conn)
			reconnectAfter = 1 * time.Second
		}
		ev, err := readEvent(bc, false, true)
		if nil != err {
			glog.Warningf("Close connection to %s for reason:%v", nc.addr, err)
			nc.close()
			return
		}
		if _, ok := ev.Msg.(*HeartBeat); ok {
			nc.hbPongTime = time.Now().Unix()
		} else {
			//TODO nothing now
		}
	}
}

func (nc *rawTCPConn) heartbeat() {
	if !nc.closed && nc.conn != nil {
		var hb HeartBeat
		req := true
		hb.Req = &req
		var ev NodeIOEvent
		ev.event = encodeNodeEvent(&hb, 0)
		ev.wal = newDiscardWAL()
		nc.write(&ev)
		if nc.hbPongTime == 0 {
			nc.hbPongTime = time.Now().Unix()
		}
	}
}

func (nc *rawTCPConn) write(ev *NodeIOEvent) {
	if nc.closed {
		ev.wal.Write(ev.event)
		return
	}
	select {
	case nc.writech <- ev:
		//do nothing
	default:
		glog.Warningf("Too slow to write events to node:%d", ev.wal.nodeId)
		ev.wal.Write(ev.event)
	}
}

func (nc *rawTCPConn) Write(p []byte) (int, error) {
	if nc.closed {
		return 0, fmt.Errorf("raw connection closed.")
	}
	ev := new(NodeIOEvent)
	ev.event = p
	ev.wal = newDiscardWAL()
	nc.write(ev)
	return len(p), nil
}

type clusterClient struct {
	nodeConnMu sync.Mutex
	allConns   map[string]*rawTCPConn
	nodeConns  []*rawTCPConn
	nodeWals   []*WAL
}

func (c *clusterClient) eraseConn(conn *rawTCPConn) {
	c.nodeConnMu.Lock()
	defer c.nodeConnMu.Unlock()
	for i, v := range c.nodeConns {
		if v == conn {
			c.nodeConns[i] = nil
		}
	}
	delete(c.allConns, conn.addr)
}

func (c *clusterClient) newNodeConn(node *Node) *rawTCPConn {
	conn := new(rawTCPConn)
	conn.addr = node.Addr
	conn.writech = make(chan *NodeIOEvent, 100000)
	conn.closech = make(chan int)
	//conn.wal = newWAL(node.Id)
	c.allConns[node.Addr] = conn
	c.nodeConns[node.Id] = conn
	go conn.readloop()
	go conn.evloop()
	return conn
}

func (c *clusterClient) getNodeConn(node *Node) (*rawTCPConn, *WAL, error) {
	c.nodeConnMu.Lock()
	defer c.nodeConnMu.Unlock()
	if len(c.nodeConns) < int(node.Id+1) {
		c.nodeConns = append(c.nodeConns, make([]*rawTCPConn, int(node.Id)+1-len(c.nodeConns))...)
	}
	var ok bool
	conn := c.nodeConns[node.Id]
	if nil == conn {
		conn, ok = c.allConns[node.Addr]
		if ok {
			c.nodeConns[node.Id] = conn
		}
	}
	if nil == conn {
		conn = c.newNodeConn(node)
	}
	var wal *WAL
	if nil != conn {
		if len(c.nodeWals) < int(node.Id+1) {
			c.nodeWals = append(c.nodeWals, make([]*WAL, int(node.Id)+1-len(c.nodeWals))...)
		}
		wal = c.nodeWals[node.Id]
		if nil == wal {
			var err error
			wal, err = newWAL(node.Id)
			if nil != err {
				return nil, nil, err
			}
			c.nodeWals[node.Id] = wal
		}
	}
	return conn, wal, nil
}

func (c *clusterClient) emitContent(hashCode uint64, content []byte) error {
	node := getNodeByHash(hashCode)
	if nil == node {
		return ErrNoNode
	}
	var ev NodeIOEvent
	ev.event = content
	conn, wal, err := c.getNodeConn(node)
	if nil != err {
		glog.Errorf("Failed to retrive connection or wal to emit event for reason:%v", err)
		return err
	}
	ev.wal = wal
	if nil != conn {
		conn.write(&ev)
	} else if nil != wal {
		ev.wal.Write(ev.event)
	}
	return nil
}

func (c *clusterClient) emit(msg proto.Message, hashCode uint64) error {
	ev := encodeNodeEvent(msg, hashCode)
	return c.emitContent(hashCode, ev)
}

func (c *clusterClient) clearInvalidConns() {
	addrSet := make(map[string]bool)
	for _, partition := range getClusterTopo().partitions {
		addrSet[partition.Addr] = true
	}
	c.nodeConnMu.Lock()
	defer c.nodeConnMu.Unlock()
	for addr, conn := range c.allConns {
		if _, ok := addrSet[addr]; !ok {
			conn.close()
		}
	}
	for i, conn := range c.nodeConns {
		if nil != conn && conn.addr != getNodeByID(int32(i)).Addr {
			c.nodeConns[i] = nil
		}
	}
}

func (c *clusterClient) checkPartitionConns() {
	var checkedAddr = make(map[string]bool)
	for _, node := range getClusterTopo().allNodes {
		if _, ok := checkedAddr[node.Addr]; !ok {
			c.getNodeConn(getNodeByID(node.Id))
			checkedAddr[node.Addr] = true
		}
	}
}

func (c *clusterClient) checkHeartbeatTimeout() {
	c.nodeConnMu.Lock()
	for addr, conn := range c.allConns {
		//close connection which have no heartbeat pong recved more than 10s
		if conn.hbPongTime > 0 && (time.Now().Unix()-conn.hbPongTime) > 10 {
			glog.Errorf("Close heartbeat timeout connection to %s", addr)
			conn.close()
		}
	}
	c.nodeConnMu.Unlock()
}
func (c *clusterClient) replayWals() {
	for i := 0; i < getClusterNodeSize(); i++ {
		node := getNodeByHash(uint64(i))
		if nil != node {
			conn, wal, err := c.getNodeConn(node)
			if nil != err {
				glog.Errorf("Failed to retrive connection to node:%d for replay event for reason:%v", node.Id, err)
				continue
			}
			if nil != wal && !wal.empty() {
				glog.Infof("Try to replay WAL for node:%d with cacehd data size:%d", node.Id, wal.cachedDataSize())
				wal.sync()
				ev := newEmptyNodeEvent()
				ev.wal = wal
				conn.write(ev)
			}
		}
	}
}

func (c *clusterClient) printWalSizes() {
	for _, wal := range c.nodeWals {
		if nil != wal && wal.cachedDataSize() > 0 {
			glog.Infof("WAL[%d] Cached Data Size:%d", wal.nodeId, wal.cachedDataSize())
		}
	}
}

func (c *clusterClient) heartbeat() {
	c.nodeConnMu.Lock()
	for _, conn := range c.allConns {
		conn.heartbeat()
	}
	c.nodeConnMu.Unlock()
}

func emit(msg proto.Message, hashCode uint64) error {
	return ssfClient.emit(msg, hashCode)
}

func emitEvent(event *Event) error {
	if len(event.raw) > 0 {
		return ssfClient.emitContent(event.GetHashCode(), event.raw)
	}
	return ssfClient.emit(event.Msg, event.GetHashCode())
}

func init() {
	ssfClient.allConns = make(map[string]*rawTCPConn)
}
