package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	xlogger "github.com/caser789/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type NodeManager interface {
	IsMaster() bool
}

var nodeManagerOnce sync.Once
var _nodeManager *nodeManager

func InitNodeManager(endpoints []string, nodeKey string) {
	nodeManagerOnce.Do(func() {
		_nodeManager = NewNodeManager(endpoints, nodeKey)
	})
}

func GetNodeManager() NodeManager {
	return _nodeManager
}

func NewNodeManager(endpoints []string, nodeKey string) *nodeManager {
	hostname, _ := os.Hostname()
	rand.Seed(int64(time.Now().Nanosecond()))
	nodeValue := fmt.Sprintf("%s_%d", hostname, rand.Int())
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second,
	})
	if err != nil {
		panic(err)
	}
	n := &nodeManager{
		cli:       cli,
		nodeKey:   nodeKey,
		nodeValue: nodeValue,
		closed:    make(chan struct{}),
	}

	go n.keepElect()
	return n
}

type nodeManager struct {
	cli *clientv3.Client

	node *node

	nodeKey   string
	nodeValue string

	closed chan struct{}
}

func (n *nodeManager) keepElect() {
	defer func() {
		n.node = nil
		xlogger.GetLogger().Info("exit campaign for no ha node")
	}()

	for {
		xlogger.GetLogger().Info("[HA]elect start", zap.String("node id", n.GetNodeId()))
		select {
		case <-n.closed:
			xlogger.GetLogger().Info("[HA]node manager done", zap.String("node id", n.GetNodeId()))
			return
		default:
		}

		// blocked to elect, elected, or error/cancelled
		err := n.elect()
		if err != nil {
			xlogger.GetLogger().Error("[HA]elect failed, sleep and re-elect", zap.Error(err), zap.String("node id", n.GetNodeId()))
			time.Sleep(time.Second)
			continue
		}

		//选举为主节点
		xlogger.GetLogger().Info("[HA]elect success", zap.String("node id", n.GetNodeId()))
		select {
		case <-n.node.Done(): // 若 node 断开，需要重新加入选举
			xlogger.GetLogger().Error("[HA]disconnected, re-elect", zap.String("node id", n.GetNodeId()))
		}
	}
}

func (n *nodeManager) GetNodeId() string {
	return n.nodeValue
}

func (n *nodeManager) Stop() {
	close(n.closed)
}

func (n *nodeManager) elect() error {
	haNode, err := newNode(n.nodeKey, n.cli)
	if err != nil {
		return err
	}

	err = haNode.Elect(n.nodeValue)
	if err != nil {
		_ = haNode.Close()
		return err
	}

	n.node = haNode
	return nil
}

func (n *nodeManager) IsMaster() bool {
	if n.node != nil {
		return n.node.IsMaster(n.nodeValue)
	}
	return false
}
