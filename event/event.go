package event

import (
	"ethattacksim/interfaces"
	"log"
)

type Event struct {
	time      int64
	targetId  string
	eventType interfaces.IEventType
}

func NewEvent(time int64, targetId string, eventType interfaces.IEventType) interfaces.IEvent {
	return &Event{time: time, targetId: targetId, eventType: eventType}
}

func (ev *Event) Type() interfaces.IEventType {
	return ev.eventType
}

func (ev *Event) TargetId() string {
	return ev.targetId
}

func (ev *Event) Time() int64 {
	return ev.time
}

func (ev *Event) Execute(world interfaces.IWorld) {
	log.Printf("event time %d\n", ev.Time())
}
