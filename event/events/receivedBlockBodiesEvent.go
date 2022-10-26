package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
)

type ReceivedBlockBodiesEvent struct {
	interfaces.IEvent
	bodies   []interfaces.IBlockBody
	senderId string
}

func NewReceivedBlockBodiesEvent(ev interfaces.IEvent, bodies []interfaces.IBlockBody, senderId string) *ReceivedBlockBodiesEvent {
	return &ReceivedBlockBodiesEvent{ev, bodies, senderId}
}

func (ev *ReceivedBlockBodiesEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_BODY_RECEIVED, ev.TargetId()), int64(len(ev.bodies)))
	logId := ""
	for i, body := range ev.bodies {
		logId += body.BlockHash()
		if i < len(ev.bodies)-1 {
			logId += ","
		}
	}
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), logId, "", node.Time())
	node.Consensus().ReceivedBlockBodiesEvent(node, ev.bodies, ev.senderId, world)
}
