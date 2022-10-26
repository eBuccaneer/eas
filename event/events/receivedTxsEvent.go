package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
)

type ReceivedTxsEvent struct {
	interfaces.IEvent
	txs      []interfaces.ITransaction
	senderId string
}

func NewReceivedTxsEvent(ev interfaces.IEvent, txs []interfaces.ITransaction, senderId string) *ReceivedTxsEvent {
	return &ReceivedTxsEvent{ev, txs, senderId}
}

func (ev *ReceivedTxsEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_TX_RECEIVED, ev.TargetId()), int64(len(ev.txs)))
	if world.SimConfig().AuditLogTxMessages() {
		txHashes := ""
		for i, tx := range ev.txs {
			txHashes += tx.Id()
			if i != len(ev.txs)-1 {
				txHashes += ","
			}
		}
		logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), txHashes, "", node.Time())
	}
	node.Consensus().ReceivedTxsEvent(node, ev.txs, ev.senderId, world)
}
