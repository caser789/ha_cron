package main

import "context"

type cronOptionKey struct{}

func injectCronOption(ctx context.Context, option *cronOption) context.Context {
	newCtx := context.WithValue(ctx, cronOptionKey{}, option)
	return newCtx
}

func getCronOption(ctx context.Context) *cronOption {
	option, ok := ctx.Value(cronOptionKey{}).(*cronOption)
	if ok {
		return option
	}
	return nil
}

type shardedTaskKey struct{}

func injectTask(ctx context.Context, task *ShardedTask) context.Context {
	newCtx := context.WithValue(ctx, shardedTaskKey{}, task)
	return newCtx
}

func getTask(ctx context.Context) *ShardedTask {
	task, ok := ctx.Value(shardedTaskKey{}).(*ShardedTask)
	if ok {
		return task
	}
	return nil
}

func getPartition(task *ShardedTask) (partition int, total int) {
	partition = -1
	if task == nil {
		return
	}

	v, ok := task.Partition[GetNodeManager().GetNodeId()]
	if ok {
		partition = v
	}
	total = len(task.Partition)
	return
}

type Metadata struct {
	Partition      int
	TotalPartition int
}
