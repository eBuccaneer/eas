package interfaces

type IEvent interface {
	Time() int64
	Type() IEventType
	TargetId() string
	// Execute executes the specific event.
	Execute(world IWorld)
}

type eventType string

type IEventType interface {
	getType() eventType
}

// this is just for preventing simple string from being used as IEventType
func (evType eventType) getType() eventType {
	return evType
}

// this is just for preventing simple string from being used as IEventType
func (evType eventType) String() string {
	return string(evType)
}

// add event types here
const (
	GENESIS_EVENT                = eventType("GenesisEvent")
	NEW_BLOCK_EVENT              = eventType("NewBlockEvent")
	NEW_BLOCK_TIMESTAMP          = eventType("NewBlockTimestamp")
	NEW_TX_EVENT                 = eventType("NewTXEvent")
	TX_CREATION_EVENT            = eventType("TxCreationEvent")
	RECEIVED_BLOCK_EVENT         = eventType("ReceivedBlockEvent")
	RECEIVED_BLOCK_HEADER_EVENT  = eventType("ReceivedBlockHeaderEvent")
	RECEIVED_BLOCK_BODIES_EVENT  = eventType("ReceivedBlockBodyEvent")
	RETRIEVE_BLOCK_HEADERS_EVENT = eventType("RetrieveBlockHeadersEvent")
	RETRIEVE_BLOCK_BODIES_EVENT  = eventType("RetrieveBlockBodiesEvent")
	RECEIVED_BLOCK_HASH_EVENT    = eventType("ReceivedBlockHashEvent")
	RECEIVED_TXS_EVENT           = eventType("ReceivedTxsEvent")
	RECEIVED_TX_HASHES_EVENT     = eventType("ReceivedTxHashesEvent")
)
