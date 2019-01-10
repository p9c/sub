package sub

import (
	"net"
)

// NodeCfg is the configuration for a Node
type NodeCfg struct {
	Listener    string
	Subscribers []string
	BufferSize  int
}

// A Node is a server with some number of subscribers
type Node struct {
	cfg         NodeCfg
	listener    *net.UDPConn
	subscribers []*net.UDPAddr
	kill        chan bool
}

// WorkerCfg is the configuration for a Worker
type WorkerCfg struct {
	Listener   string
	Node       string
	BufferSize int
}

// A Worker is a node that subscribes to a Node's messages
type Worker struct {
	cfg      WorkerCfg
	listener *net.UDPConn
	node     *net.UDPAddr
	kill     chan bool
}

const (
	uNet = "udp4"
)
