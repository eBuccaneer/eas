package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	ti "time"
)

type NewBlockEvent struct {
	interfaces.IEvent
	block       interfaces.IBlock
	miningStart int64
}

func NewNewBlockEvent(ev interfaces.IEvent, block interfaces.IBlock, miningStart int64) *NewBlockEvent {
	return &NewBlockEvent{ev, block, miningStart}
}

func (ev *NewBlockEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Timer(interfaces.METRIC_BLOCK_CREATED.String(), ti.Duration(node.Time()-ev.miningStart))
	metrics.Timer(interfaces.METRIC_BLOCK_GAS_USED.String(), ti.Duration(ev.block.Header().GasUsed()))
	metrics.Timer(interfaces.METRIC_BLOCK_GAS_LIMIT.String(), ti.Duration(ev.block.Header().GasLimit()))
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_CREATED, ev.TargetId()), 1)
	logger.AuditEvent(node.Id(), ev.Type(), ev.block.Hash(), "", node.Time())
	logger.AuditEvent(node.Id(), interfaces.NEW_BLOCK_TIMESTAMP, ev.block.Hash(), "", ev.block.Header().Time()*1000000000)
	node.Consensus().NewBlockEvent(node, ev.block, world, ev.Time())
}
