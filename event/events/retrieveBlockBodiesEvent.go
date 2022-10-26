package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"strings"
)

type RetrieveBlockBodiesEvent struct {
	interfaces.IEvent
	hashes   []string
	senderId string
}

func NewRetrieveBlockBodiesEvent(ev interfaces.IEvent, hashes []string, senderId string) *RetrieveBlockBodiesEvent {
	return &RetrieveBlockBodiesEvent{ev, hashes, senderId}
}

func (ev *RetrieveBlockBodiesEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_BODY_RETRIEVAL, ev.TargetId()), 1)
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), strings.Join(ev.hashes, ","), "", node.Time())
	node.Consensus().RetrieveBlockBodiesEvent(node, ev.hashes, ev.senderId, world)
}
