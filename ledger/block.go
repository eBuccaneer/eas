package ledger

import "ethattacksim/interfaces"

type Block struct {
	BHeader          interfaces.IBlockHeader `json:"h"`
	BBody            interfaces.IBlockBody   `json:"b"`
	BTotalDifficulty int                     `json:"d"`
}

type BlockHeader struct {
	BHash       string `json:"h"`
	BTxHash     string `json:"th"`
	BUncleHash  string `json:"u"`
	BParentHash string `json:"p"`
	BMinerId    string `json:"m"` // aka coinbase
	BDifficulty int    `json:"d"`
	BGasUsed    int    `json:"g"`
	BGasLimit   int    `json:"l"`
	BTime       int64  `json:"t"`
	BNumber     int    `json:"n"`
	BSize       int    `json:"s"`
	BValid      bool   `json:"v"`
}

type BlockBody struct {
	blockHash    string
	transactions []interfaces.ITransaction
	uncles       []interfaces.IBlockHeader
	BValid       bool `json:"v"`
	BtxCount     int  `json:"t"`
}

type Transaction struct {
	id                        string `json:"h"`
	nonce                     int
	senderId                  string
	gasUsed                   int
	gasPrice                  int     // gwei
	TValid                    bool    `json:"v"`
	specialTxStateComputation float64 // special tx state computation delay for attacks
}

func NewBlock(header interfaces.IBlockHeader, body interfaces.IBlockBody, totalDifficulty int) interfaces.IBlock {
	return &Block{header, body, totalDifficulty}
}

func NewBlockHeader(hash string, txHash string, uncleHash string, parentHash string, minerId string, difficulty int, gasUsed int, gasLimit int, time int64, height int, size int, valid bool) interfaces.IBlockHeader {
	return &BlockHeader{hash, txHash, uncleHash, parentHash, minerId, difficulty, gasUsed, gasLimit, time, height, size, valid}
}

func NewBlockBody(blockHash string, txs []interfaces.ITransaction, uncles []interfaces.IBlockHeader, valid bool, txCount int) interfaces.IBlockBody {
	return &BlockBody{blockHash, txs, uncles, valid, txCount}
}

func NewTx(id string, nonce int, senderId string, gasUsed int, gasPrice int, valid bool, specialTxStateComputation float64) interfaces.ITransaction {
	return &Transaction{id, nonce, senderId, gasUsed, gasPrice, valid, specialTxStateComputation}
}

func (block *Block) Hash() string {
	return block.BHeader.Hash()
}

func (block *Block) Header() interfaces.IBlockHeader {
	return block.BHeader
}

func (block *Block) Body() interfaces.IBlockBody {
	return block.BBody
}

func (block *Block) TotalDifficulty() int {
	return block.BTotalDifficulty
}

func (block *Block) SetTotalDifficulty(td int) {
	block.BTotalDifficulty = td
}

func (block *Block) ParentHash() string {
	return block.BHeader.ParentHash()
}

func (body *BlockBody) BlockHash() string {
	return body.blockHash
}

func (body *BlockBody) Transactions() []interfaces.ITransaction {
	return body.transactions
}

func (body *BlockBody) Uncles() []interfaces.IBlockHeader {
	return body.uncles
}

func (body *BlockBody) AddTransaction(tx ...interfaces.ITransaction) {
	body.transactions = append(body.transactions, tx...)
}

func (body *BlockBody) AddUncles(uncles ...interfaces.IBlockHeader) {
	body.uncles = append(body.uncles, uncles...)
}

func (body *BlockBody) IsValid() bool {
	return body.BValid
}

func (header *BlockHeader) Hash() string {
	return header.BHash
}

func (header *BlockHeader) UncleHash() string {
	return header.BUncleHash
}

func (header *BlockHeader) TxHash() string {
	return header.BTxHash
}

func (header *BlockHeader) ParentHash() string {
	return header.BParentHash
}

func (header *BlockHeader) MinerId() string {
	return header.BMinerId
}

func (header *BlockHeader) Time() int64 {
	return header.BTime
}

func (header *BlockHeader) Number() int {
	return header.BNumber
}

func (header *BlockHeader) Size() int {
	return header.BSize
}

func (header *BlockHeader) SetNumber(number int) {
	header.BNumber = number
}

func (header *BlockHeader) Difficulty() int {
	return header.BDifficulty
}

func (header *BlockHeader) GasUsed() int {
	return header.BGasUsed
}

func (header *BlockHeader) GasLimit() int {
	return header.BGasLimit
}

func (header *BlockHeader) IsValid() bool {
	return header.BValid
}

func (tx *Transaction) Id() string {
	return tx.id
}

func (tx *Transaction) Nonce() int {
	return tx.nonce
}

func (tx *Transaction) SenderId() string {
	return tx.senderId
}

func (tx *Transaction) GasUsed() int {
	return tx.gasUsed
}

func (tx *Transaction) GasPrice() int {
	return tx.gasPrice
}

func (tx *Transaction) IsValid() bool {
	return tx.TValid
}

func (tx *Transaction) SpecialTxStateComputation() float64 {
	return tx.specialTxStateComputation
}
