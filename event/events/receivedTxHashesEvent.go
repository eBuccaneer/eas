package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"strings"
)

type ReceivedTxHashesEvent struct {
	interfaces.IEvent
	txHashes []string
	senderId string
}

func NewReceivedTxHashesEvent(ev interfaces.IEvent, txHashes []string, senderId string) *ReceivedTxHashesEvent {
	return &ReceivedTxHashesEvent{ev, txHashes, senderId}
}

func (ev *ReceivedTxHashesEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	if world.SimConfig().AuditLogTxMessages() {
		logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), strings.Join(ev.txHashes, ","), "", node.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_TX_HASH_RECEIVED, ev.TargetId()), int64(len(ev.txHashes)))
	node.Consensus().ReceivedTxHashesEvent(node, ev.txHashes, ev.senderId, world)
}
