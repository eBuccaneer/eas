package node

import (
	"ethattacksim/interfaces"
)

type Node struct {
	NId        string  `json:"i"`
	NHashPower float64 `json:"p"`
	NCpuPower  float64 `json:"v"`
	NIsOnline  bool    `json:"o"`
	time       int64
	peers      []interfaces.INode
	NNodeType  interfaces.INodeType `json:"t"`
	NLocation  interfaces.ILocation `json:"loc"`
	NLedger    interfaces.ILedger   `json:"l"`
	network    interfaces.INetwork
	consensus  interfaces.IConsensus
	nonce      int
}

func NewNode(id string, hashPower float64, cpuPower float64, nodeType interfaces.INodeType, location interfaces.ILocation, ledger interfaces.ILedger, network interfaces.INetwork, consensus interfaces.IConsensus) interfaces.INode {
	return &Node{id, hashPower, cpuPower, true, 0, make([]interfaces.INode, 0, 10), nodeType, location, ledger, network, consensus, 0}
}

func (node *Node) HashPower() float64 {
	return node.NHashPower
}

func (node *Node) CpuPower() float64 {
	return node.NCpuPower
}

func (node *Node) IsOnline() bool {
	return node.NIsOnline
}

func (node *Node) SetOnline(isOnline bool) {
	node.NIsOnline = isOnline
}

func (node *Node) Time() int64 {
	return node.time
}

func (node *Node) IncrementTime(time int64) {
	node.time += time
}

func (node *Node) SetTime(time int64) {
	node.time = time
}

func (node *Node) Id() string {
	return node.NId
}

func (node *Node) Peers() []interfaces.INode {
	return node.peers
}

func (node *Node) AddPeers(peer ...interfaces.INode) {
	node.peers = append(node.peers, peer...)
}

func (node *Node) AddPeersToFront(peer ...interfaces.INode) {
	node.peers = append(peer, node.peers...)
}

func (node *Node) RemovePeer(peerId string) {
	for i, peer := range node.peers {
		if peer.Id() == peerId {
			node.peers = append(node.peers[:i], node.peers[i+1:]...)
			break
		}
	}
}

func (node *Node) Type() interfaces.INodeType {
	return node.NNodeType
}

func (node *Node) Location() interfaces.ILocation {
	return node.NLocation
}

func (node *Node) Ledger() interfaces.ILedger {
	return node.NLedger
}

func (node *Node) Consensus() interfaces.IConsensus {
	return node.consensus
}

func (node *Node) Network() interfaces.INetwork {
	return node.network
}

func (node *Node) Nonce() int {
	return node.nonce
}

func (node *Node) IncNonce() {
	node.nonce++
}
