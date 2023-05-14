package main

import "encoding/json"

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
