package interfaces

type INetwork interface {
	BroadcastBlock(block IBlock, node INode, world IWorld, targets ...INode)
	BroadcastBlockHash(hash string, number int, node INode, world IWorld, targets ...INode)
	RetrieveBlockHeaders(node INode, peer INode, world IWorld, originBlockHash string, num int, reverse bool, skip int)
	RetrieveBlockBodies(node INode, peer INode, world IWorld, hashes []string)
	SendBlockHeaders(node INode, peer INode, world IWorld, headers []IBlockHeader)
	SendBlockBodies(node INode, peer INode, world IWorld, bodies []IBlockBody)
	BroadcastTxs(transaction []ITransaction, node INode, world IWorld, targets ...INode)
	BroadcastTxHashes(txHashes []string, node INode, world IWorld, targets ...INode)
	RetrieveTxs(txHashes []string, node INode, peer INode, world IWorld)
	MaxPeers() int
	ConnectToPeers(nodeId string, world IWorld)
	DropPeer(node INode, peerId string, world IWorld)
}
