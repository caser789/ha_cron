package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	xlogger "github.com/caser789/logger"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

var once sync.Once
var _taskManager *taskManager

const (
	TASK_RESULT_COMPLETED = "completed"
	TASK_RESULT_ERROR     = "error"
	TASK_RESULT_PANIC     = "panic"
)

func GetTaskManager() *taskManager {
	once.Do(func() {
		_taskManager = NewTaskManager()
	})
	return _taskManager
}

type taskManager struct {
	c         *cron.Cron
	taskCount uint32
}

func init() {
	xlogger.InitLogger(&xlogger.Config{
		Level:         xlogger.DebugLvl,
		PrintToStdout: true,
		SplitLevel:    xlogger.SplitNone,
	})
}

func NewTaskManager() *taskManager {
	c := cron.New()
	c.Start()

	return &taskManager{
		c: c,
	}
}

type cronOption struct {
	name string
}

func (m *taskManager) Register(spec string, cmd func(ctx context.Context) error) {
	defaultTaskName := fmt.Sprintf("task_%d", atomic.AddUint32(&m.taskCount, 1))
	ops := &cronOption{name: defaultTaskName}

	m.c.AddFunc(spec, func() {
		task(cmd, ops)
	})
}

func task(cmd func(ctx context.Context) error, option *cronOption) {
	opName := fmt.Sprintf("timer_task_%s", option.name)
	taskResult := TASK_RESULT_COMPLETED

	defer func() {
		if err := recover(); err != nil {
			xlogger.GetLogger().Error("run task error", zap.Error(err.(error)), zap.String("task name", opName), zap.String("result", taskResult))
			taskResult = TASK_RESULT_PANIC
		}
	}()

	xlogger.GetLogger().Info("start to run task", zap.String("task name", opName))

	err := cmd(context.Background())
	if err != nil {
		taskResult = TASK_RESULT_ERROR
		xlogger.GetLogger().Error("run task error", zap.Error(err), zap.String("task name", opName), zap.String("result", taskResult))
	}

	xlogger.GetLogger().Info("complete to run task", zap.String("task name", opName), zap.String("result", taskResult))
}

func main() {
	fmt.Println("--------------------------------------------------")
	t := GetTaskManager()

	spec := "1 * * * * *"
	t.Register(spec, func(ctx context.Context) error {
		fmt.Printf("run task @%s\n", time.Now())
		return nil
	})

	select {}
	fmt.Println("==================================================")
}
