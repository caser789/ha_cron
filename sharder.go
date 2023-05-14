package main

import (
	"context"
	"encoding/json"
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

type Event struct {
	EventType int    `json:"event_type"`
	EventId   string `json:"event_id"`
	//毫秒时间戳
	CreateTime uint64 `json:"create_time"`
	Key        string `json:"key"`
	Value      string `json:"value"`
}

func (e *Event) String() string {
	res, _ := json.Marshal(e)
	return string(res)
}

type EventProcessor func(*Event) error

var (
	taskEventProcessor EventProcessor
)
