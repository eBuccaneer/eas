package ledger

import (
	"ethattacksim/interfaces"
	"log"
	"sort"
	"strings"
)

type Ledger struct {
	current                map[string]interfaces.IBlock
	ledger                 map[string]interfaces.IBlock
	state                  map[string]bool
	LCurrentLedgerByHeight []interfaces.IBlock `json:"c"`
	headHash               string
	txQueue                map[string]interfaces.ITransaction
	possibleUncles         map[string]interfaces.IBlockHeader // possible uncle block headers
	uncles                 map[string]bool                    // uncle blocks that are included in the current ledger
}

func NewLedger() interfaces.ILedger {
	return &Ledger{make(map[string]interfaces.IBlock), make(map[string]interfaces.IBlock), make(map[string]bool), make([]interfaces.IBlock, 0, 100), "", make(map[string]interfaces.ITransaction, 100), make(map[string]interfaces.IBlockHeader, 2), make(map[string]bool, 100)}
}

func (ledger *Ledger) Get() map[string]interfaces.IBlock {
	return ledger.ledger
}

func (ledger *Ledger) GetCurrent() map[string]interfaces.IBlock {
	return ledger.current
}

func (ledger *Ledger) State() map[string]bool {
	return ledger.state
}

func (ledger *Ledger) CurrentLedgerByHeight() []interfaces.IBlock {
	return ledger.LCurrentLedgerByHeight
}

func (ledger *Ledger) SetCurrentLedgerByHeight(ledgerByHeight []interfaces.IBlock) {
	ledger.LCurrentLedgerByHeight = ledgerByHeight
}

func (ledger *Ledger) PossibleUncles() map[string]interfaces.IBlockHeader {
	return ledger.possibleUncles
}

func (ledger *Ledger) Uncles() map[string]bool {
	return ledger.uncles
}

func (ledger *Ledger) Reorg(node interfaces.INode, newHead interfaces.IBlock, world interfaces.IWorld) bool {
	// check if parents are the same retrieved by number and hash
	parentFromHead := node.Ledger().CurrentLedgerByHeight()[node.Ledger().GetBlock(node, newHead.ParentHash()).Header().Number()]
	parentFromNumber := node.Ledger().CurrentLedgerByHeight()[newHead.Header().Number()-1]
	if parentFromHead.Hash() == parentFromNumber.Hash() {
		// split into remaining chain and chain of blocks that has to be removed from current ledger
		ledgerByHeightRemove := node.Ledger().CurrentLedgerByHeight()[parentFromHead.Header().Number()+1:]
		ledgerByHeightNew := node.Ledger().CurrentLedgerByHeight()[:parentFromHead.Header().Number()+1]

		// handle blocks to remove
		for _, oldBlock := range ledgerByHeightRemove {
			if _, ok := node.Ledger().GetCurrent()[oldBlock.Hash()]; !ok {
				log.Panic("remove but not in ledger " + oldBlock.Hash())
			}
			// delete block from current ledger
			delete(node.Ledger().GetCurrent(), oldBlock.Hash())
			for _, uncle := range oldBlock.Body().Uncles() {
				// delete every uncle contained in block from current ledger
				delete(node.Ledger().Uncles(), uncle.Hash())
				// add removed uncles to possible uncles for future mining if dist not too far
				if uncle.Number()+int(world.SimConfig().MaxUncleDist()) >= node.Ledger().Length(node)-1 {
					node.Ledger().PossibleUncles()[uncle.Hash()] = uncle
				}
			}
			// add removed block itself to possible uncles if dist is not too far
			if oldBlock.Header().Number()+int(world.SimConfig().MaxUncleDist()) >= node.Ledger().Length(node)-1 {
				node.Ledger().PossibleUncles()[oldBlock.Hash()] = oldBlock.Header()
			}
			for _, tx := range oldBlock.Body().Transactions() {
				// delete transactions contained in block from current ledger
				//delete(node.Ledger().Txs(), tx.Id())
				// add removed txs to queue
				//  here it is assumed that all tx are valid again after reorg
				if !strings.HasPrefix(tx.Id(), "R") {
					// the prefix indicates it was randomly created and can be tossed away
					node.Ledger().AddTxsToQueue(node, tx)
				}
			}
		}
		// add new head to current ledger
		node.Ledger().SetCurrentLedgerByHeight(append(ledgerByHeightNew, newHead))
		node.Ledger().GetCurrent()[newHead.Hash()] = newHead
		node.Ledger().Get()[newHead.Hash()] = newHead
		node.Ledger().State()[newHead.Hash()] = true
		node.Ledger().SetHeadHash(newHead.Hash())
		// add uncles of new block and delete them from possible uncles
		for _, uncle := range newHead.Body().Uncles() {
			node.Ledger().Uncles()[uncle.Hash()] = true
			delete(node.Ledger().PossibleUncles(), uncle.Hash())
		}
		delete(node.Ledger().PossibleUncles(), newHead.Hash())
		// add txs of new block to current ledger
		node.Ledger().AddTxs(node, newHead.Body().Transactions()...)
		return true
	}
	return false
}

func (ledger *Ledger) AppendBlockToCurrent(node interfaces.INode, block interfaces.IBlock) {
	node.Ledger().Get()[block.Hash()] = block
	node.Ledger().GetCurrent()[block.Hash()] = block
	node.Ledger().SetCurrentLedgerByHeight(append(node.Ledger().CurrentLedgerByHeight(), block))
	node.Ledger().SetHeadHash(block.Hash())
	node.Ledger().State()[block.Hash()] = true
	for _, uncle := range block.Body().Uncles() {
		node.Ledger().Uncles()[uncle.Hash()] = true
		delete(node.Ledger().PossibleUncles(), uncle.Hash())
	}
	delete(node.Ledger().PossibleUncles(), block.Hash())
	node.Ledger().AddTxs(node, block.Body().Transactions()...)
}

func (ledger *Ledger) WriteBlock(node interfaces.INode, block interfaces.IBlock, withState bool) {
	node.Ledger().Get()[block.Hash()] = block
	node.Ledger().State()[block.Hash()] = withState
	node.Ledger().PossibleUncles()[block.Hash()] = block.Header()
}

func (ledger *Ledger) GetBlock(node interfaces.INode, hash string) interfaces.IBlock {
	return node.Ledger().Get()[hash]
}

func (ledger *Ledger) HasBlock(node interfaces.INode, hash string) bool {
	_, exists := node.Ledger().Get()[hash]
	return exists
}

func (ledger *Ledger) CurrentHasBlock(node interfaces.INode, hash string) bool {
	_, exists := node.Ledger().GetCurrent()[hash]
	return exists
}

func (ledger *Ledger) CurrentGetBlockByNumber(node interfaces.INode, number int) interfaces.IBlock {
	if len(node.Ledger().CurrentLedgerByHeight()) > number {
		return node.Ledger().CurrentLedgerByHeight()[number]
	} else {
		return nil
	}
}

func (ledger *Ledger) Head(node interfaces.INode) interfaces.IBlock {
	return node.Ledger().Get()[node.Ledger().HeadHash()]
}

func (ledger *Ledger) HeadHash() string {
	return ledger.headHash
}

func (ledger *Ledger) SetHeadHash(headHash string) {
	ledger.headHash = headHash
}

func (ledger *Ledger) AddTxsToQueue(node interfaces.INode, txs ...interfaces.ITransaction) {
	for _, tx := range txs {
		node.Ledger().QueuedTxs()[tx.Id()] = tx
	}
}

func (ledger *Ledger) QueuedTxs() map[string]interfaces.ITransaction {
	return ledger.txQueue
}

// also removes from queue if present
func (ledger *Ledger) AddTxs(node interfaces.INode, txs ...interfaces.ITransaction) {
	for _, tx := range txs {
		if _, exists := node.Ledger().QueuedTxs()[tx.Id()]; exists {
			delete(node.Ledger().QueuedTxs(), tx.Id())
		}
	}
}

func (ledger *Ledger) KnowsQueuedTx(node interfaces.INode, hash string) bool {
	_, existsQueue := node.Ledger().QueuedTxs()[hash]
	return existsQueue
}

func (ledger *Ledger) SortedTxByLocalAndRemote(node interfaces.INode) ([]interfaces.ITransaction, []interfaces.ITransaction) {
	// nonce sorting is not done, implement if necessary
	locals := make([]interfaces.ITransaction, 0)
	remotes := make([]interfaces.ITransaction, 0, 50)
	for _, tx := range node.Ledger().QueuedTxs() {
		if tx.SenderId() == node.Id() {
			locals = append(locals, tx)
		} else {
			remotes = append(remotes, tx)
		}
	}
	sort.Slice(locals, func(i, j int) bool {
		return locals[i].GasPrice() > locals[j].GasPrice()
	})
	sort.Slice(remotes, func(i, j int) bool {
		return remotes[i].GasPrice() > remotes[j].GasPrice()
	})
	return locals, remotes
}

func (ledger *Ledger) Length(node interfaces.INode) int {
	return len(node.Ledger().GetCurrent())
}

func (ledger *Ledger) GetTx(node interfaces.INode, hash string) interfaces.ITransaction {
	return node.Ledger().QueuedTxs()[hash]
}
