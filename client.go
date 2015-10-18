package ssf

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"net"
	"sync"
	"time"
)

var ssfClient clusterClient

type NodeIOEvent struct {
	wal   *WAL
	event []byte
}

func newNodeEvent(msg proto.Message, hashCode uint64) *NodeIOEvent {
	var event Event
	event.HashCode = hashCode
	event.MsgType = int32(GetEventType(msg))
	event.Msg = msg
	var buf bytes.Buffer
	writeEvent(&event, &buf)
	ev := new(NodeIOEvent)
	ev.event = buf.Bytes()

	return ev
}

func newEmptyNodeEvent() *NodeIOEvent {
	ev := new(NodeIOEvent)
	ev.event = make([]byte, 0)
	return ev
}

// nodeConn wraps a connection, usually a persistent one
type rawTcpConn struct {
	addr       string
	conn       net.Conn
	writech    chan *NodeIOEvent
	closech    chan int
	closed     bool
	hbPongTime int64 //the time recv heatbeat pong
}

func (nc *rawTcpConn) close() {
	if !nc.closed {
		nc.closed = true
		nc.closech <- 1
	}
}

func (nc *rawTcpConn) logUnsentEvents() {
	for {
		select {
		case ev := <-nc.writech:
			ev.wal.Write(ev.event)
		default:
			return
		}
	}
}

func (nc *rawTcpConn) evloop() {
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
					go ev.wal.replay(nc)
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

func (nc *rawTcpConn) readloop() {
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
		ev, err := readEvent(bc)
		if nil != err {
			glog.Warningf("Close connection to %s for reason:%v", nc.addr, err)
			nc.close()
			return
		}
		if ev.MsgType == int32(EventType_EVENT_HEARTBEAT) {
			nc.hbPongTime = time.Now().Unix()
		}
		//ssfClient.process()
		//nc.write(ev)
	}
}

func (nc *rawTcpConn) heartbeat() {
	if !nc.closed && nc.conn != nil {
		var hb HeartBeat
		req := true
		hb.Req = &req
		ev := newNodeEvent(&hb, 0)
		ev.wal = newDiscardWAL()
		nc.write(ev)
		if nc.hbPongTime == 0 {
			nc.hbPongTime = time.Now().Unix()
		}
	}
}

func (nc *rawTcpConn) write(ev *NodeIOEvent) {
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

func (nc *rawTcpConn) Write(p []byte) (int, error) {
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
	allConns   map[string]*rawTcpConn
	nodeConns  []*rawTcpConn
	nodeWals   []*WAL
}

func (c *clusterClient) eraseConn(conn *rawTcpConn) {
	c.nodeConnMu.Lock()
	defer c.nodeConnMu.Unlock()
	for i, v := range c.nodeConns {
		if v == conn {
			c.nodeConns[i] = nil
		}
	}
	delete(c.allConns, conn.addr)
}

func (c *clusterClient) newNodeConn(node *Node) *rawTcpConn {
	conn := new(rawTcpConn)
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

func (c *clusterClient) getNodeConn(node *Node) (*rawTcpConn, *WAL, error) {
	c.nodeConnMu.Lock()
	defer c.nodeConnMu.Unlock()
	if len(c.nodeConns) < int(node.Id+1) {
		c.nodeConns = append(c.nodeConns, make([]*rawTcpConn, int(node.Id)+1-len(c.nodeConns))...)
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

func (c *clusterClient) emit(msg proto.Message, hashCode uint64) {
	node := getNodeByHash(hashCode)
	if nil == node {
		return
	}
	if isSelfNode(node) {
		var event Event
		event.HashCode = hashCode
		event.Msg = msg
		event.MsgType = GetEventType(msg)
		ssfCfg.Handler.OnEvent(&event)
		return
	}
	ev := newNodeEvent(msg, hashCode)
	conn, wal, err := c.getNodeConn(node)
	if nil != err {
		glog.Errorf("Failed to retrive connection or wal to emit event for reason:%v", err)
		return
	}
	ev.wal = wal
	if nil != conn {
		conn.write(ev)
		return
	}
	if nil != wal {
		ev.wal.Write(ev.event)
	}
}

func (c *clusterClient) checkPartitionConns() {
	for _, part := range topo.partitions {
		c.getNodeConn(getNodeById(part.Nodes[0]))
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
	for _, wal := range c.nodeWals {
		if nil != wal && !wal.empty() {
			ev := newEmptyNodeEvent()
			ev.wal = wal
			conn, _, err := c.getNodeConn(getNodeById(wal.nodeId))
			if nil != err {
				glog.Errorf("Failed to retrive connection or wal to emit event for reason:%v", err)
			} else {
				conn.write(ev)
			}
		}
	}
}

func (c *clusterClient) routine() {
	hbTickChan := time.NewTicker(time.Millisecond * 1000).C
	checkTickChan := time.NewTicker(time.Millisecond * 5000).C
	for {
		select {
		case <-hbTickChan:
			c.checkPartitionConns()
			c.nodeConnMu.Lock()
			for _, conn := range c.allConns {
				conn.heartbeat()
			}
			c.nodeConnMu.Unlock()
		case <-checkTickChan:
			c.checkHeartbeatTimeout()
			c.replayWals()
		}
	}
}

func Emit(msg proto.Message, hashCode uint64) {
	ssfClient.emit(msg, hashCode)
}

func init() {
	ssfClient.allConns = make(map[string]*rawTcpConn)
	go ssfClient.routine()
}
