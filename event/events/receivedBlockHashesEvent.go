package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"strings"
)

type ReceivedBlockHashesEvent struct {
	interfaces.IEvent
	hashes   []string
	numbers  []int
	senderId string
}

func NewReceivedBlockHashesEvent(ev interfaces.IEvent, hashes []string, numbers []int, senderId string) *ReceivedBlockHashesEvent {
	return &ReceivedBlockHashesEvent{ev, hashes, numbers, senderId}
}

func (ev *ReceivedBlockHashesEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_HASH_RECEIVED, ev.TargetId()), int64(len(ev.hashes)))
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), strings.Join(ev.hashes, ","), "", node.Time())
	node.Consensus().ReceivedBlockHashesEvent(node, ev.hashes, ev.numbers, ev.senderId, world)
}
