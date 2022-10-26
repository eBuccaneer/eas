package interfaces

import (
	"errors"
)

type IConsensus interface {
	BlockSeen() map[string]map[string]bool
	TxSeen() map[string]map[string]bool
	RetrievingHeaders() map[string]bool
	RetrievingBodies() map[string]IBlockHeader
	FutureBlockQueue() map[string]IBlock
	FutureBlockQueueSenderId() map[string]string
	ReceivedBlockEvent(node INode, block IBlock, senderId string, world IWorld)
	NewBlockEvent(node INode, block IBlock, world IWorld, evTime int64)
	ReceivedBlockHashesEvent(node INode, hashes []string, numbers []int, senderId string, world IWorld)
	RetrieveBlockHeadersEvent(node INode, originBlockHash string, num int, reverse bool, skip int, senderId string, world IWorld)
	ReceivedTxsEvent(node INode, txs []ITransaction, senderId string, world IWorld)
	ReceivedTxHashesEvent(node INode, txHashes []string, senderId string, world IWorld)
	RetrieveTxsEvent(node INode, txHashes []string, senderId string, world IWorld)
	ReceivedBlockHeadersEvent(node INode, headers []IBlockHeader, senderId string, world IWorld)
	RetrieveBlockBodiesEvent(node INode, hashes []string, senderId string, world IWorld)
	ReceivedBlockBodiesEvent(node INode, bodies []IBlockBody, senderId string, world IWorld)
	// InsertBlock processes a new block.
	// It returns if a new head was written and if the process was ok.
	InsertBlock(block IBlock, node INode, ledger ILedger, world IWorld, peerId string, evTime int64) (newHead bool, ok bool)
	// InsertToSidechain inserts a block to the sidechain and only imports to current chain if td > localTd.
	// It returns if a new head was written and if the process was ok.
	InsertToSidechain(blocks []IBlock, headerErrors []error, node INode, ledger ILedger, world IWorld) (newHead bool, ok bool)
	// InsertToChain inserts a block to the ledger (only subsequent block numbers allowed.
	// It returns if a new head was written and if the process was ok.
	InsertToChain(blocks []IBlock, node INode, ledger ILedger, world IWorld) (newHead bool, ok bool)
	CheckReorg(localTd int, externalTd int, block IBlock, currentHead IBlock, node INode) bool
	WriteBlock(block IBlock, ledger ILedger, node INode, auditPrefix string)
	AppendBlock(block IBlock, ledger ILedger, node INode, auditPrefix string)
	ReorgChain(block IBlock, ledger ILedger, node INode, world IWorld, auditPrefix string) bool
	VerifyHeader(block IBlock, ledger ILedger, world IWorld, node INode, isUncle bool) (timeConsumed int64, err error)
	// VerifyHeaders verifies headers in parallel.
	VerifyHeaders(blocks []IBlock, ledger ILedger, world IWorld, node INode) (errors []error)
	VerifyBody(block IBlock, ledger ILedger, world IWorld, node INode) (err error)
	VerifyState(block IBlock, node INode, checkPastTx bool) (ok bool)
	VerifyTx(tx ITransaction, node INode) (ok bool)
	CalcDifficulty(parentHeader IBlockHeader, time int64, world IWorld) int
	TotalDifficulty(node INode, hash string, ledger ILedger) int
	MarkBlockSeen(node INode, hash string, peerId string)
	MarkTxSeen(node INode, hash string, peerId string)
	RetrieveHeaders(node INode, ledger ILedger, originBlockHash string, num int, reverse bool, skip int) []IBlockHeader
	RetrieveBodies(node INode, ledger ILedger, hashes []string) []IBlockBody
	BroadcastNewBlockTargets(node INode, block IBlock, propagate bool, excludeIds ...string) (targets []INode)
	BroadcastReceivedBlockTargets(node INode, block IBlock, propagate bool, excludeIds ...string) (targets []INode)
	BroadcastTxTargets(node INode, tx ITransaction, propagate bool, excludeIds ...string) (targets []INode)
	MineBlock(ledger ILedger, node INode, world IWorld)
	GetTxsForBlock(gasUsed int, txs []ITransaction, gasLimit int) (transactions []ITransaction, gasAmount int)
	// GetGasLimit returns the gas limit for the new block.
	GetGasLimit(currentHead IBlockHeader, world IWorld) (gasLimit int)
}

var (
	ErrUnknownAncestor = errors.New("unknown ancestor")
	ErrPrunedAncestor  = errors.New("pruned ancestor")
	ErrFutureBlock     = errors.New("block in the future")
	ErrKnownBlock      = errors.New("known block")
	ErrDanglingUncle   = errors.New("dangling uncle")
	ErrInvalidHeader   = errors.New("invalid header")
	ErrInvalidBody     = errors.New("invalid body")
	ErrUncleIsAncestor = errors.New("uncle is ancestor")
	ErrOlderBlock      = errors.New("older than parent")
)
