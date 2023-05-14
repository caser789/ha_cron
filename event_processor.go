package main

import (
	"context"
	"encoding/json"
	"fmt"

	xlogger "github.com/caser789/logger"
	"go.uber.org/zap"
)

type EventProcessor func(*Event) error

var (
	taskEventProcessor EventProcessor
)

func processTaskEvent(event *Event) error {
	if event.EventType != EVENT_TYPE_TASK {
		return fmt.Errorf("event type error")
	}
	xlogger.GetLogger().Info("process task event", zap.Any("event value", event.Value))

	shardedTask := new(ShardedTask)
	err := json.Unmarshal([]byte(event.Value), shardedTask)
	if err != nil {
		xlogger.GetLogger().Error("parse sharded task error", zap.Error(err), zap.String("event value", event.Value))
		return err
	}

	partition, totalPartition := getPartition(shardedTask)
	option := &cronOption{
		masterOnly: false,
		name:       shardedTask.TaskName,
	}
	// opName := fmt.Sprintf("timer_task_%s", shardedTask.TaskName)
	ctx := context.Background()
	ctx = injectTask(ctx, shardedTask)

	taskResult := TASK_RESULT_COMPLETED
	defer func() {
		if err := recover(); err != nil {
			xlogger.GetLogger().Error("start to run task error", zap.Error(err.(error)), zap.String("task name", shardedTask.TaskName), zap.String("node id", GetNodeManager().GetNodeId()), zap.Int("partition", partition), zap.Int("total", totalPartition))
			taskResult = TASK_RESULT_PANIC
		}
	}()

	xlogger.GetLogger().Info("start to run task", zap.String("task name", option.name), zap.String("node id", GetNodeManager().GetNodeId()), zap.Int("partition", partition), zap.Int("total", totalPartition))
	cmd := getShardedTask(shardedTask.TaskName)
	if cmd != nil {
		err = cmd(ctx, Metadata{Partition: partition, TotalPartition: totalPartition})
	} else {
		err = fmt.Errorf("not found task:%s", shardedTask.TaskName)
	}
	if err != nil {
		taskResult = TASK_RESULT_ERROR
		xlogger.GetLogger().Info("sharded task failed", zap.Error(err), zap.String("task name", option.name), zap.String("node id", GetNodeManager().GetNodeId()), zap.String("result", taskResult), zap.Int("partition", partition), zap.Int("total", totalPartition))
		return err
	}
	xlogger.GetLogger().Info("sharded task completed", zap.String("task name", option.name), zap.String("node id", GetNodeManager().GetNodeId()), zap.String("result", taskResult), zap.Int("partition", partition), zap.Int("total", totalPartition))
	return err
}
