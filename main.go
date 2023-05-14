package main

import (
	"context"
	"fmt"
	xlogger "github.com/caser789/logger"
	"github.com/robfig/cron"
	"go.uber.org/zap"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var once sync.Once
var _taskManager *taskManager

const (
	TASK_RESULT_COMPLETED = "completed"
	TASK_RESULT_ERROR     = "error"
	TASK_RESULT_PANIC     = "panic"
)

func GetTaskManager(nodeKey string, endpoints []string) *taskManager {
	once.Do(func() {
		_taskManager = NewTaskManager(nodeKey, endpoints)
	})
	return _taskManager
}

type taskManager struct {
	c         *cron.Cron
	taskCount uint32

	nodeKey   string
	endpoints []string

	once sync.Once
}

func init() {
	xlogger.InitLogger(&xlogger.Config{
		Level:         xlogger.DebugLvl,
		PrintToStdout: true,
		SplitLevel:    xlogger.SplitNone,
	})
}

func NewTaskManager(nodeKey string, endpoints []string) *taskManager {
	c := cron.New()
	c.Start()

	return &taskManager{
		c:         c,
		nodeKey:   nodeKey,
		endpoints: endpoints,
	}
}

type cronOption struct {
	name       string
	masterOnly bool
	isSharded  bool
}

type CronOption func(*cronOption)

func WithMasterOnly() CronOption {
	return func(option *cronOption) {
		option.masterOnly = true
	}
}

func WithSharedTask() CronOption {
	return func(option *cronOption) {
		option.isSharded = true
		option.masterOnly = true
	}
}

type CmdFun func(ctx context.Context, metadata Metadata) error

func (m *taskManager) Register(spec string, fn CmdFun, opts ...CronOption) {
	defaultTaskName := fmt.Sprintf("task_%d", atomic.AddUint32(&m.taskCount, 1))
	ops := &cronOption{name: defaultTaskName}
	for _, f := range opts {
		f(ops)
	}

	if ops.isSharded {
		InitNodeManager(m.endpoints, m.nodeKey)
		registerShardedTask(fn, ops)
		m.c.AddFunc(spec, func() {
			task(triggerTask, ops)
		})
		return
	}

	if ops.masterOnly {
		InitNodeManager(m.endpoints, m.nodeKey)
	}

	_ = m.c.AddFunc(spec, func() {
		task(fn, ops)
	})
}

func task(fn CmdFun, option *cronOption) {
	opName := fmt.Sprintf("timer_task_%s", option.name)
	taskResult := TASK_RESULT_COMPLETED

	ctx := context.Background()
	ctx = injectCronOption(ctx, option)

	defer func() {
		if err := recover(); err != nil {
			xlogger.GetLogger().Error("run task error", zap.Error(err.(error)), zap.String("task name", opName), zap.String("result", taskResult))
			taskResult = TASK_RESULT_PANIC
		}
	}()

	if option.masterOnly && !GetNodeManager().IsMaster() {
		xlogger.GetLogger().Info("skip for not leader", zap.String("task name", opName))
		return
	}

	xlogger.GetLogger().Info("start to run task", zap.String("task name", opName))
	err := fn(ctx, Metadata{Partition: 0, TotalPartition: 1})
	if err != nil {
		taskResult = TASK_RESULT_ERROR
		xlogger.GetLogger().Error("run task error", zap.Error(err), zap.String("task name", opName), zap.String("result", taskResult))
	}
	xlogger.GetLogger().Info("complete to run task", zap.String("task name", opName), zap.String("result", taskResult))
}

func main() {
	fmt.Println("--------------------------------------------------")
	nodeKey := os.Getenv("NODE_KEY")
	endpoints := os.Getenv("ENDPOINTS")

	t := GetTaskManager(nodeKey, strings.Split(endpoints, ","))

	spec := "1 * * * * *"
	t.Register(spec, func(ctx context.Context, metadata Metadata) error {
		for i := 0; i < 10; i++ {
			if i%metadata.TotalPartition != metadata.Partition {
				continue
			}
			fmt.Printf("run task %d\n", i)
		}
		return nil
	}, WithSharedTask())

	select {}
	fmt.Println("==================================================")
}
