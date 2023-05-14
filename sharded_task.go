package main

import (
	"encoding/json"
	"time"
)

type ShardedTask struct {
	TaskName  string
	TaskId    string // 默认用 traceId
	Partition map[string]int
}

func (t *ShardedTask) toEvent() *Event {
	event := new(Event)
	event.EventType = EVENT_TYPE_TASK
	event.EventId = t.TaskId
	event.Key = t.TaskName
	event.CreateTime = uint64(time.Now().UnixNano() / 1000000)
	v, _ := json.Marshal(t)
	event.Value = string(v)
	return event
}
