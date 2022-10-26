package interfaces

type ILedger interface {
	Get() map[string]IBlock
	// GetCurrent returns the current best chain.
	GetCurrent() map[string]IBlock
	// CurrentLedgerByHeight returns the current best chain by height.
	CurrentLedgerByHeight() []IBlock
	SetCurrentLedgerByHeight([]IBlock)
	State() map[string]bool
	PossibleUncles() map[string]IBlockHeader
	Uncles() map[string]bool
	Reorg(node INode, newHead IBlock, world IWorld) bool
	// AppendBlockToCurrent writes to both current and normal ledger with state.
	AppendBlockToCurrent(node INode, block IBlock)
	WriteBlock(node INode, block IBlock, withState bool)
	GetBlock(node INode, hash string) IBlock
	HasBlock(node INode, hash string) bool
	CurrentHasBlock(node INode, hash string) bool
	CurrentGetBlockByNumber(node INode, number int) IBlock
	Head(node INode) IBlock
	HeadHash() string
	SetHeadHash(headHash string)
	AddTxsToQueue(node INode, txs ...ITransaction)
	QueuedTxs() map[string]ITransaction
	AddTxs(node INode, txs ...ITransaction)
	SortedTxByLocalAndRemote(node INode) ([]ITransaction, []ITransaction)
	KnowsQueuedTx(node INode, hash string) bool
	GetTx(node INode, hash string) ITransaction
	Length(node INode) int
}

type IBlock interface {
	Header() IBlockHeader
	Body() IBlockBody
	TotalDifficulty() int
	SetTotalDifficulty(td int)
	ParentHash() string
	Hash() string
}

type IBlockHeader interface {
	Hash() string
	TxHash() string
	UncleHash() string
	ParentHash() string
	Difficulty() int
	GasUsed() int
	GasLimit() int
	MinerId() string
	Time() int64
	Number() int
	Size() int
	SetNumber(number int)
	IsValid() bool // instead of really computing verification
}

type IBlockBody interface {
	BlockHash() string
	Transactions() []ITransaction
	Uncles() []IBlockHeader
	AddTransaction(tx ...ITransaction)
	AddUncles(uncles ...IBlockHeader)
	IsValid() bool // instead of really computing verification
}

type ITransaction interface {
	Id() string
	Nonce() int // is not used in simulator for now, implement if necessary
	SenderId() string
	GasUsed() int
	GasPrice() int
	IsValid() bool             // instead of really computing verification
	SpecialTxStateComputation() float64 // special tx state computation delay for attacks
}
