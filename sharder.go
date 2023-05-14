package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	xlogger "github.com/caser789/logger"
	"go.uber.org/zap"
)

const (
	EVENT_TYPE_TASK = 1
)

func (n *nodeManager) eventLoop() error {
	ctx := context.TODO()
	for {
		eventChan := n.watchEvent(ctx)
		for event := range eventChan {
			switch event.EventType {
			case EVENT_TYPE_TASK:
				if taskEventProcessor != nil {
					go taskEventProcessor(&event)
				}
			default:
				xlogger.GetLogger().Error("event type invalid", zap.String("node id", n.GetNodeId()), zap.String("event", event.String()))
			}
		}
	}
}

func (n *nodeManager) watchEvent(ctx context.Context) <-chan Event {
	eventChan := make(chan Event, 1)
	msgKey := n.nodeKey + "_event"
	rspChan := n.cli.Watch(ctx, msgKey)
	go func() {
		defer close(eventChan)
		for rsp := range rspChan {
			for _, etcdEvent := range rsp.Events {
				event := new(Event)
				err := json.Unmarshal(etcdEvent.Kv.Value, event)
				if err != nil {
					xlogger.GetLogger().Error("parse event failed", zap.Error(err))
					continue
				}

				select {
				case eventChan <- *event:
				case <-time.After(time.Second):
					xlogger.GetLogger().Error("put event chan timeout")
				}
			}
		}
	}()
	return eventChan
}

var (
	shardedTaskMap  map[string]CmdFun
	shardedTaskLock sync.Mutex
)

func init() {
	shardedTaskMap = make(map[string]CmdFun)
	taskEventProcessor = processTaskEvent
}

func registerShardedTask(fn CmdFun, option *cronOption) {
	shardedTaskLock.Lock()
	defer shardedTaskLock.Unlock()

	shardedTaskMap[option.name] = fn
}

func getShardedTask(taskName string) CmdFun {
	shardedTaskLock.Lock()
	defer shardedTaskLock.Unlock()

	cmd, ok := shardedTaskMap[taskName]
	if ok {
		return cmd
	}
	return nil
}

func triggerTask(ctx context.Context, metadata Metadata) error {
	option := getCronOption(ctx)
	shardedTask := new(ShardedTask)
	shardedTask.TaskName = option.name
	shardedTask.TaskId = fmt.Sprintf("%d", rand.Int31n(1000000))
	shardedTask.Partition = make(map[string]int)
	_, nodeIds := GetNodeManager().GetAllNodeId()
	for i, nodeId := range nodeIds {
		shardedTask.Partition[nodeId] = i
	}
	event := shardedTask.toEvent()
	xlogger.GetLogger().Info("before pub event", zap.Any("task", shardedTask))
	err := GetNodeManager().PubEvent(context.TODO(), event)
	return err
}
