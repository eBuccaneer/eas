package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
)

type ReceivedBlockEvent struct {
	interfaces.IEvent
	block    interfaces.IBlock
	senderId string
}

func NewReceivedBlockEvent(ev interfaces.IEvent, block interfaces.IBlock, senderId string) *ReceivedBlockEvent {
	return &ReceivedBlockEvent{ev, block, senderId}
}

func (ev *ReceivedBlockEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_RECEIVED, ev.TargetId()), 1)
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), ev.block.Hash(), "", node.Time())
	node.Consensus().ReceivedBlockEvent(node, ev.block, ev.senderId, world)
}
