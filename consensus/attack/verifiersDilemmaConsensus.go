package attack

import (
	"ethattacksim/interfaces"
)

type VerifiersDilemmaConsensus struct {
	interfaces.IConsensus
}

func NewVerifiersDilemmaConsensus(consensus interfaces.IConsensus) interfaces.IConsensus {
	return &VerifiersDilemmaConsensus{IConsensus: consensus}
}

func (c *VerifiersDilemmaConsensus) VerifyTx(tx interfaces.ITransaction, node interfaces.INode) (ok bool) {
	return true
}

func (c *VerifiersDilemmaConsensus) VerifyState(block interfaces.IBlock, node interfaces.INode, checkPastTx bool) (ok bool) {
	return true
}

func (c *VerifiersDilemmaConsensus) BroadcastReceivedBlockTargets(node interfaces.INode, block interfaces.IBlock, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	return
}
