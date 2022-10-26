package network

import (
	"ethattacksim/event"
	"ethattacksim/event/events"
	"ethattacksim/interfaces"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"ethattacksim/util/random"
	"fmt"
	"strings"
	ti "time"
)

type Network struct {
	maxPeers int
}

func NewNetwork(maxPeers int) interfaces.INetwork {
	return &Network{maxPeers}
}

func (n *Network) BroadcastBlock(block interfaces.IBlock, node interfaces.INode, world interfaces.IWorld, targets ...interfaces.INode) {
	sendStart := node.Time()
	for _, peer := range targets {
		latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), block.Header().Size())
		eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), block.Header().Size())
		ev := events.NewReceivedBlockEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_BLOCK_EVENT), block, node.Id())
		//node.IncrementTime(latSend) // increment node time for sending? is node busy?
		logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), block.Hash(), "", sendStart+latSend)
		world.Queue().Add(ev)
		metrics.Timer(interfaces.METRIC_BLOCK_SENT.String(), ti.Duration(eventTime-sendStart))
		sendStart += latSend
	}
}

func (n *Network) BroadcastBlockHash(hash string, number int, node interfaces.INode, world interfaces.IWorld, targets ...interfaces.INode) {
	sendStart := node.Time()
	for _, peer := range targets {
		messageSize := world.SimConfig().Sizes()["hash"]
		latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
		eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
		ev := events.NewReceivedBlockHashesEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_BLOCK_HASH_EVENT), []string{hash}, []int{number}, node.Id())
		//node.IncrementTime(latSend) // increment node time for sending? is node busy?
		logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), hash, "", sendStart+latSend)
		world.Queue().Add(ev)
		metrics.Timer(interfaces.METRIC_BLOCK_HASH_SENT.String(), ti.Duration(eventTime-sendStart))
		sendStart += latSend
	}
}

func (n *Network) RetrieveBlockHeaders(node interfaces.INode, peer interfaces.INode, world interfaces.IWorld, originBlockHash string, num int, reverse bool, skip int) {
	messageSize := world.SimConfig().Sizes()["getHeaders"]
	sendStart := node.Time()
	latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
	eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
	ev := events.NewRetrieveBlockHeadersEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RETRIEVE_BLOCK_HEADERS_EVENT), originBlockHash, num, reverse, skip, node.Id())
	//node.IncrementTime(latSend) // increment node time for sending? is node busy?
	logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), fmt.Sprintf("originHash:%v,num:%v,reverse:%v,skip%v", originBlockHash, num, reverse, skip), "", sendStart+latSend)
	world.Queue().Add(ev)
	metrics.Timer(interfaces.METRIC_BLOCK_HEADER_RETRIEVAL.String(), ti.Duration(eventTime-sendStart))
}

func (n *Network) SendBlockHeaders(node interfaces.INode, peer interfaces.INode, world interfaces.IWorld, headers []interfaces.IBlockHeader) {
	messageSize := world.SimConfig().Sizes()["header"] * len(headers)
	sendStart := node.Time()
	latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
	eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
	ev := events.NewReceivedBlockHeadersEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_BLOCK_HEADER_EVENT), headers, node.Id())
	logId := ""
	for i, header := range headers {
		logId += header.Hash()
		if i < len(headers)-1 {
			logId += ","
		}
	}
	//node.IncrementTime(latSend) // increment node time for sending? is node busy?
	logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), logId, "", sendStart+latSend)
	world.Queue().Add(ev)
	metrics.Timer(interfaces.METRIC_BLOCK_HEADER_RECEIVED.String(), ti.Duration(eventTime-sendStart))
}

func (n *Network) RetrieveBlockBodies(node interfaces.INode, peer interfaces.INode, world interfaces.IWorld, hashes []string) {
	messageSize := world.SimConfig().Sizes()["hash"] * len(hashes)
	sendStart := node.Time()
	latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
	eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
	ev := events.NewRetrieveBlockBodiesEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RETRIEVE_BLOCK_BODIES_EVENT), hashes, node.Id())
	//node.IncrementTime(latSend) // increment node time for sending? is node busy?
	logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), strings.Join(hashes, ","), "", sendStart+latSend)
	world.Queue().Add(ev)
	metrics.Timer(interfaces.METRIC_BLOCK_BODY_RETRIEVAL.String(), ti.Duration(eventTime-sendStart))
}

func (n *Network) SendBlockBodies(node interfaces.INode, peer interfaces.INode, world interfaces.IWorld, bodies []interfaces.IBlockBody) {
	txCount := 0
	uncleCount := 0
	logId := ""
	for i, body := range bodies {
		txCount += len(body.Transactions())
		uncleCount += len(body.Uncles())
		logId += body.BlockHash()
		if i < len(bodies)-1 {
			logId += ","
		}
	}
	messageSize := world.SimConfig().Sizes()["tx"]*txCount + world.SimConfig().Sizes()["header"]*uncleCount
	sendStart := node.Time()
	latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
	eventTime := node.Time() + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
	ev := events.NewReceivedBlockBodiesEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_BLOCK_BODIES_EVENT), bodies, node.Id())
	//node.IncrementTime(latSend) // increment node time for sending? is node busy?
	logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), logId, "", sendStart+latSend)
	world.Queue().Add(ev)
	metrics.Timer(interfaces.METRIC_BLOCK_BODY_RECEIVED.String(), ti.Duration(eventTime-sendStart))
}

func (n *Network) BroadcastTxs(transactions []interfaces.ITransaction, node interfaces.INode, world interfaces.IWorld, targets ...interfaces.INode) {
	sendStart := node.Time()
	for _, peer := range targets {
		messageSize := world.SimConfig().Sizes()["tx"] * len(transactions)
		latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
		eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
		ev := events.NewReceivedTxsEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_TXS_EVENT), transactions, node.Id())
		//node.IncrementTime(latSend) // increment node time for sending? is node busy?
		if world.SimConfig().AuditLogTxMessages() {
			txHashes := ""
			for i, tx := range transactions {
				txHashes += tx.Id()
				if i != len(transactions)-1 {
					txHashes += ","
				}
			}
			logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), txHashes, "", sendStart+latSend)
		}
		world.Queue().Add(ev)
		metrics.Timer(interfaces.METRIC_TX_SENT.String(), ti.Duration(eventTime-sendStart))
		sendStart += latSend
	}
}

func (n *Network) BroadcastTxHashes(txHashes []string, node interfaces.INode, world interfaces.IWorld, targets ...interfaces.INode) {
	sendStart := node.Time()
	for _, peer := range targets {
		messageSize := world.SimConfig().Sizes()["hash"] * len(txHashes)
		latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
		eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
		ev := events.NewReceivedTxHashesEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_TX_HASHES_EVENT), txHashes, node.Id())
		//node.IncrementTime(latSend) // increment node time for sending? is node busy?
		if world.SimConfig().AuditLogTxMessages() {
			logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), strings.Join(txHashes, ","), "", sendStart+latSend)
		}
		world.Queue().Add(ev)
		metrics.Timer(interfaces.METRIC_TX_HASH_SENT.String(), ti.Duration(eventTime-sendStart))
		sendStart += latSend
	}
}

func (n *Network) RetrieveTxs(txHashes []string, node interfaces.INode, peer interfaces.INode, world interfaces.IWorld) {
	messageSize := world.SimConfig().Sizes()["hash"] * len(txHashes)
	sendStart := node.Time()
	latSend := random.Latency(node.Location(), peer.Location()) + random.SendThroughput(node.Location(), peer.Location(), messageSize)
	eventTime := sendStart + latSend + random.ReceiveThroughput(node.Location(), peer.Location(), messageSize)
	ev := events.NewRetrieveTxsEventEvent(event.NewEvent(eventTime, peer.Id(), interfaces.RECEIVED_BLOCK_HASH_EVENT), txHashes, node.Id())
	//node.IncrementTime(latSend) // increment node time for sending? is node busy?
	if world.SimConfig().AuditLogTxMessages() {
		logger.AuditEventSent(node.Id(), peer.Id(), ev.Type(), strings.Join(txHashes, ","), "", sendStart+latSend)
	}
	world.Queue().Add(ev)
	metrics.Timer(interfaces.METRIC_TX_RETRIEVAL.String(), ti.Duration(eventTime-sendStart))
}

func (n *Network) MaxPeers() int {
	return n.maxPeers
}

func (n *Network) ConnectToPeers(nodeId string, world interfaces.IWorld) {
	// TODO: better peer connection algorithm respecting maxPeerCount
	localNode := world.Nodes()[nodeId]
	for i := 0; i < localNode.Network().MaxPeers()*2; i++ {
		if len(localNode.Peers()) >= localNode.Network().MaxPeers() {
			break
		}
		remoteNode := world.Nodes()[PeerOracle(nodeId, world.NodeIds(), world.Nodes())]
		if remoteNode.Network().MaxPeers() > len(remoteNode.Peers()) {
			if !ContainsPeer(localNode, remoteNode) {
				metrics.Counter(metrics.NameFormat(interfaces.METRIC_PEER_ADDED, localNode.Id()), 1)
				metrics.Counter(interfaces.METRIC_PEER_ADDED.String(), 1)
				localNode.AddPeersToFront(remoteNode) // add "outgoing" peers to front of slice
			}
			if !ContainsPeer(remoteNode, localNode) {
				metrics.Counter(metrics.NameFormat(interfaces.METRIC_PEER_ADDED, remoteNode.Id()), 1)
				metrics.Counter(interfaces.METRIC_PEER_ADDED.String(), 1)
				remoteNode.AddPeers(localNode) // add "ingoing" peers to end of slice
			}
		}
	}
}

// drops the peer and finds a new one
func (n *Network) DropPeer(node interfaces.INode, peerId string, world interfaces.IWorld) {
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_PEER_DROPPED, node.Id()), 1)
	metrics.Counter(interfaces.METRIC_PEER_DROPPED.String(), 1)
	node.RemovePeer(peerId)
	world.Nodes()[peerId].RemovePeer(node.Id())
	for i := 0; i < 50; i++ {
		// max 50 tries to find one new peer
		newPeerId := PeerOracle(node.Id(), world.NodeIds(), world.Nodes())
		remoteNode := world.Nodes()[newPeerId]
		if ContainsPeer(node, remoteNode) {
			continue
		}
		if remoteNode.Network().MaxPeers() > len(remoteNode.Peers()) {
			node.AddPeersToFront(remoteNode) // add "outgoing" peers to front of slice
			if !ContainsPeer(remoteNode, node) {
				remoteNode.AddPeers(node) // add "ingoing" peers to end of slice
			}
			metrics.Counter(metrics.NameFormat(interfaces.METRIC_PEER_ADDED, node.Id()), 1)
			metrics.Counter(interfaces.METRIC_PEER_ADDED.String(), 1)
			break
		}
	}
}

func PeerOracle(nodeId string, nodeIds []string, nodes map[string]interfaces.INode) (selectedPeerId string) {
	selectedPeerId = nodeId
	tried := 0
	for selectedPeerId == nodeId {
		tried++
		i := int(random.Uniform() * float64(len(nodeIds)))
		peer := nodes[nodeIds[i]]
		if len(peer.Peers()) < peer.Network().MaxPeers() {
			selectedPeerId = peer.Id()
		}
		if tried > 200 {
			// just to prevent endless loops in rare cases
			break
		}
	}
	return
}

func ContainsPeer(n1 interfaces.INode, n2 interfaces.INode) bool {
	for _, p := range n1.Peers() {
		if p.Id() == n2.Id() {
			return true
		}
	}
	return false
}
