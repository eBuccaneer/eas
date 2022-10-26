package events

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"fmt"
)

type RetrieveBlockHeadersEvent struct {
	interfaces.IEvent
	originBlockHash string
	num             int
	reverse         bool
	skip            int
	senderId        string
}

func NewRetrieveBlockHeadersEvent(ev interfaces.IEvent, originBlockHash string, num int, reverse bool, skip int, senderId string) *RetrieveBlockHeadersEvent {
	return &RetrieveBlockHeadersEvent{ev, originBlockHash, num, reverse, skip, senderId}
}

func (ev *RetrieveBlockHeadersEvent) Execute(world interfaces.IWorld) {
	node := world.Nodes()[ev.TargetId()]
	if ev.Time() > node.Time() {
		node.SetTime(ev.Time())
	}
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_HEADER_RETRIEVAL, ev.TargetId()), 1)
	logger.AuditEventReceived(ev.TargetId(), ev.senderId, ev.Type(), fmt.Sprintf("originHash:%v,num:%v,reverse:%v,skip%v", ev.originBlockHash, ev.num, ev.reverse, ev.skip), "", node.Time())
	node.Consensus().RetrieveBlockHeadersEvent(node, ev.originBlockHash, ev.num, ev.reverse, ev.skip, ev.senderId, world)
}
