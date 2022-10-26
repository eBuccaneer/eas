package interfaces

type IConfig interface {
	Seed() uint64
	UseMetrics() bool
	UsePprof() bool
	OutPath() string
	PrintLogToConsole() bool
	PrintAuditLogToConsole() bool
	PrintMemStats() bool
	EndTime() int64
	NodeCount() uint64
	AuditLogTxMessages() bool
	SimulateTransactionCreation() bool
	CheckPastTxWhenVerifyingState() bool
	MaxUncleDist() uint64
	TxPerMin() uint64
	BombDelay() uint64
	OverallHashPower() float64
	MiningPoolsHashPower() []float64
	MiningPoolsCpuPower() []float64
	Limits() map[string]int
	Sizes() map[string]int
	AttackerActive() bool
	Attacker() IAttackerConfig
}

type IAttackerConfig interface {
	Type() string
	HashPower() []float64
	MaxPeers() []int
	CpuPower() []float64
	Location() []string
	Numbers() map[string]float64
	Strings() map[string]string
}

type metricName string

type IMetricName interface {
	getMetricName() metricName
	String() string
}

type IRNG interface {
	Rand() float64
}

// this is just for preventing simple string from being used as IMetricName
func (mName metricName) getMetricName() metricName {
	return mName
}

func (mName metricName) String() string {
	return string(mName)
}

// add metric names here
const (
	METRIC_BLOCK_CREATED          = metricName("BlockCreated")
	METRIC_BLOCK_GAS_USED         = metricName("BlockGasUsed")
	METRIC_BLOCK_GAS_LIMIT        = metricName("BlockGasLimit")
	METRIC_TX_CREATED             = metricName("TxCreated")
	METRIC_BLOCK_RECEIVED         = metricName("BlockReceived")
	METRIC_BLOCK_HEADER_RECEIVED  = metricName("BlockHeaderReceived")
	METRIC_BLOCK_BODY_RECEIVED    = metricName("BlockBodyReceived")
	METRIC_BLOCK_HASH_RECEIVED    = metricName("BlockHashReceived")
	METRIC_BLOCK_HEADER_RETRIEVAL = metricName("BlockHeaderRetrieval")
	METRIC_BLOCK_BODY_RETRIEVAL   = metricName("BlockBodyRetrieval")
	METRIC_BLOCK_INSERT  		  = metricName("BlockInsert")
	METRIC_TX_GAS                 = metricName("TxGas")
	METRIC_TX_PRICE               = metricName("TxPrice")
	METRIC_TX_RETRIEVAL           = metricName("TxRetrieval")
	METRIC_TX_RECEIVED            = metricName("TxReceived")
	METRIC_TX_SENT                = metricName("TxSent")
	METRIC_TX_HASH_RECEIVED       = metricName("TxHashReceived")
	METRIC_TX_HASH_SENT           = metricName("TxHashSent")
	METRIC_BLOCK_SENT             = metricName("BlockSent")
	METRIC_BLOCK_HASH_SENT        = metricName("BlockHashSent")
	METRIC_BLOCK_APPENDED         = metricName("BlockAppended")
	METRIC_BLOCK_WRITTEN          = metricName("BlockWritten")
	METRIC_BLOCK_WRITTEN_REORG    = metricName("BlockWrittenReorg")
	METRIC_PEER_DROPPED           = metricName("PeerDropped")
	METRIC_PEER_ADDED             = metricName("PeerAdded")
	METRIC_EVENT_REAL_TIME        = metricName("EventRealTime")
)
