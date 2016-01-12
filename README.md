# SSF
Simple streaming process framework

# Dependency
- [glog](https://github.com/golang/glog)
- [golang protobuf](https://github.com/golang/protobuf)
- [gogo protobuf](https://github.com/gogo/protobuf)
- [mmap-go](https://github.com/edsrzf/mmap-go)
- [go-zookeeper](https://github.com/samuel/go-zookeeper)
- [gotoolkit](https://github.com/yinqiwen/gotoolkit)

# Features
- Two Cluster Mode Support
    + Static Multi Servers(no zookeeper dependency)
    + Dynamic Multi Servers(with zookeeper dependency)
- Standalone Process Task 
    + All tasks is running as standalone process, which communicate framework over unix sockets.
    + Multi language support
- Write Ahead Log
    + Cache data when cluster do failover work or some cluster node is temporary unavailable. 
- Online Trouble Shooting
    + Use 'telent' to attach running process and eneter 'otsc' to enter online trouble shooting interactive mode.

# Disvantages
- No message delivery guarantees
    + One-way message delivery between nodes. 
    + Message may lost when cluster failover, process restart, etc.

# Example

- [Word Count](https://github.com/yinqiwen/ssf/tree/master/examples/wc)



