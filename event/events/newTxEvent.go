package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	ti "time"
)

/**
event that mimics the network (some nodes) creating transactions
*/
type NewTxEvent struct {
	interfaces.IEvent
	tx       interfaces.ITransaction
	senderId string
}

func NewNewTxEvent(ev interfaces.IEvent, tx interfaces.ITransaction, senderId string) *NewTxEvent {
	return &NewTxEvent{ev, tx, senderId}
}

func (ev *NewTxEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(interfaces.METRIC_TX_CREATED.String(), 1)
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_TX_RECEIVED, ev.TargetId()), 1)
	metrics.Timer(interfaces.METRIC_TX_GAS.String(), ti.Duration(ev.tx.GasUsed()))
	metrics.Timer(interfaces.METRIC_TX_PRICE.String(), ti.Duration(ev.tx.GasPrice()))
	if world.SimConfig().AuditLogTxMessages() {
		logger.AuditEvent(node.Id(), ev.Type(), ev.tx.Id(), "", node.Time())
	}
	node.Consensus().ReceivedTxsEvent(node, []interfaces.ITransaction{ev.tx}, ev.senderId, world)
}
