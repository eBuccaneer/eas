package interfaces

type INode interface {
	HashPower() float64
	CpuPower() float64
	Time() int64
	IncrementTime(time int64)
	SetTime(time int64)
	IsOnline() bool
	SetOnline(isOnline bool)
	Id() string
	Peers() []INode
	// AddPeers is a helper method that adds peers to the end of peer slice (data structure specific)
	AddPeers(peer ...INode)
	// AddPeersToFront is a helper method that adds peers to front of peer slice (data structure specific)
	AddPeersToFront(peer ...INode)
	RemovePeer(peerId string)
	Location() ILocation
	Type() INodeType
	Ledger() ILedger
	Consensus() IConsensus
	Network() INetwork
	Nonce() int
	IncNonce()
}

type nodeType string

type INodeType interface {
	getNodeType() nodeType
}

// this is just for preventing simple string from being used as INodeType
func (nType nodeType) getNodeType() nodeType {
	return nType
}

// add node types here
const (
	FULL_NODE = nodeType("FullNode")
	ATTACKER_NODE = nodeType("AttackerNode")
)

type location string

type ILocation interface {
	getLocation() location
	String() string
}

// this is just for preventing simple string from being used as location
func (l location) getLocation() location {
	return l
}

func (l location) String() string {
	return string(l)
}

// add locations here
const (
	TOKIO   = location("Tokio")
	IRELAND = location("Ireland")
	OHIO    = location("Ohio")
)

var LOCATION_MAP = map[interface{}]ILocation{
	"Tokio":   TOKIO,
	"Ireland": IRELAND,
	"Ohio":    OHIO,
	"":        TOKIO,
	nil:       OHIO,
}
