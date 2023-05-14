package main

import (
	"context"
	"time"

	xlogger "github.com/caser789/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

type node struct {
	session  *concurrency.Session
	election *concurrency.Election
}

func newNode(nodeKey string, cli *clientv3.Client) (*node, error) {
	session, err := concurrency.NewSession(cli)
	if err != nil {
		return nil, err
	}

	election := concurrency.NewElection(session, nodeKey)
	haNode := &node{
		session:  session,
		election: election,
	}
	return haNode, nil
}

// Elect puts a value as eligible for the election. It blocks until
// it is elected, an error occurs, or the context is cancelled.
func (n *node) Elect(nodeValue string) error {
	return n.election.Campaign(context.Background(), nodeValue)
}

func (n *node) Done() <-chan struct{} {
	return n.session.Done()
}

func (n *node) Close() error {
	if n.session != nil {
		return n.session.Close()
	}
	return nil
}

func (n *node) IsMaster(nodeValue string) bool {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	leader, err := n.election.Leader(ctx)
	if err == concurrency.ErrElectionNoLeader {
		return false
	}

	if err != nil {
		xlogger.GetLogger().Error("is_master_error", zap.Error(err), zap.String("node_value", nodeValue))
		return false
	}

	if len(leader.Kvs) == 0 {
		return false
	}

	return string(leader.Kvs[0].Value) == nodeValue
}
