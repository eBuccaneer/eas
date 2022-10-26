package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"strings"
)

type RetrieveTxsEvent struct {
	interfaces.IEvent
	txHashes []string
	senderId string
}

func NewRetrieveTxsEventEvent(ev interfaces.IEvent, txHashes []string, senderId string) *RetrieveTxsEvent {
	return &RetrieveTxsEvent{ev, txHashes, senderId}
}

func (ev *RetrieveTxsEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_TX_RETRIEVAL, ev.TargetId()), int64(len(ev.txHashes)))
	if world.SimConfig().AuditLogTxMessages() {
		logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), strings.Join(ev.txHashes, ","), "", node.Time())
	}
	node.Consensus().RetrieveTxsEvent(node, ev.txHashes, ev.senderId, world)
}
