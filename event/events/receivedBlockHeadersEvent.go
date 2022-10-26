package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
)

type ReceivedBlockHeadersEvent struct {
	interfaces.IEvent
	headers  []interfaces.IBlockHeader
	senderId string
}

func NewReceivedBlockHeadersEvent(ev interfaces.IEvent, headers []interfaces.IBlockHeader, senderId string) *ReceivedBlockHeadersEvent {
	return &ReceivedBlockHeadersEvent{ev, headers, senderId}
}

func (ev *ReceivedBlockHeadersEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_HEADER_RECEIVED, ev.TargetId()), int64(len(ev.headers)))
	headerIds := ""
	for i, header := range ev.headers {
		headerIds += header.Hash()
		if i != len(ev.headers)-1 {
			headerIds += ","
		}
	}
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), headerIds, "", node.Time())
	node.Consensus().ReceivedBlockHeadersEvent(node, ev.headers, ev.senderId, world)
}
