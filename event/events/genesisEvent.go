package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
)

type GenesisEvent struct {
	interfaces.IEvent
	block interfaces.IBlock
}

func NewGenesisEvent(ev interfaces.IEvent, block interfaces.IBlock) *GenesisEvent {
	return &GenesisEvent{ev, block}
}

func (ev *GenesisEvent) Execute(world interfaces.IWorld) {
	// this event starts the node for simulation
	logger.AuditEvent(ev.TargetId(), ev.Type(), "", "", ev.Time())
	node := world.Nodes()[ev.TargetId()]
	if node.IsOnline() {
		if ev.Time() > node.Time() {
			node.SetTime(ev.Time())
		}
		node.Ledger().AppendBlockToCurrent(node, ev.block)
		node.Consensus().MineBlock(node.Ledger(), node, world)
	}
}
