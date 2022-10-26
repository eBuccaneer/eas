package attack

import (
	"crypto/sha256"
	"encoding/base64"
	"ethattacksim/event"
	"ethattacksim/event/events"
	"ethattacksim/interfaces"
	ledg "ethattacksim/ledger"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"ethattacksim/util/random"
	"math"
	"sort"
	"strings"
	ti "time"
)

type SelfishMiningConsensus struct {
	interfaces.IConsensus
	blocksAhead     []interfaces.IBlock
	blockHandledNum map[int]bool
}

func NewSelfishMiningConsensus(consensus interfaces.IConsensus) interfaces.IConsensus {
	return &SelfishMiningConsensus{IConsensus: consensus, blocksAhead: make([]interfaces.IBlock, 0), blockHandledNum: make(map[int]bool)}
}

func (c *SelfishMiningConsensus) InsertBlock(block interfaces.IBlock, node interfaces.INode, ledger interfaces.ILedger, world interfaces.IWorld, peerId string, evTime int64) (newHead bool, ok bool) {
	startTime := node.Time()
	selfishRange := 100 // otherwise it would be possible to not check blocks if far ahead with selfish mining
	switch {
	case block.Header().Number() > ledger.Length(node)+selfishRange: // ledger.Length() == headBlock number + 1
		node.Consensus().FutureBlockQueue()[block.ParentHash()] = block
		node.Consensus().FutureBlockQueueSenderId()[block.Hash()] = peerId
		logger.Audit(node.Id(), "FUTURE_BLOCK", block.Hash(), "", node.Time())
		return false, false
	case block.Header().Number()+int(world.SimConfig().MaxUncleDist())+selfishRange < ledger.Length(node)-1:
		// block should be dismissed because it's too old
		logger.Audit(node.Id(), "BLOCK_OLD", block.Hash(), "", node.Time())
		return false, false
	case ledger.HasBlock(node, block.Hash()):
		// block should be dismissed because already in chain
		logger.Audit(node.Id(), "BLOCK_KNOWN", block.Hash(), "", node.Time())
		return false, false
	case !ledger.HasBlock(node, block.ParentHash()):
		// block should be dismissed because unknown parent and not future
		logger.Audit(node.Id(), "UNKNOWN_PARENT", block.Hash(), "", node.Time())
		return false, false
	default:
		if peerId != node.Id() { // self called with possible uncle block otherwise
			_, err := node.Consensus().VerifyHeader(block, ledger, world, node, false)
			switch err {
			case nil:
				broadcastPropagateTargets := node.Consensus().BroadcastReceivedBlockTargets(node, block, true, node.Id())
				node.Network().BroadcastBlock(block, node, world, broadcastPropagateTargets...)
			// not modelled
			/*case interfaces.ErrFutureBlock:
			// do nothing*/
			default:
				logger.Audit(node.Id(), "INVALID_HEADER", block.Hash(), err.Error(), node.Time())
				node.Network().DropPeer(node, peerId, world)
				return false, false
			}
		}
		newHead, ok := node.Consensus().InsertToChain([]interfaces.IBlock{block}, node, ledger, world)
		if !ok {
			logger.Audit(node.Id(), "IMPORT_FAILED", block.Hash(), "", node.Time())
			node.Network().DropPeer(node, peerId, world)
			return false, false
		}
		if peerId != node.Id() { // self called with possible uncle block otherwise
			broadcastTargets := node.Consensus().BroadcastReceivedBlockTargets(node, block, false, node.Id())
			node.Network().BroadcastBlockHash(block.Hash(), block.Header().Number(), node, world, broadcastTargets...)
			if newHead {
				if newEvent := world.Queue().DeleteOneOfTypeForNode(interfaces.NEW_BLOCK_EVENT, node); newEvent != nil {
					newEvent.Execute(world)
				}
				node.Consensus().MineBlock(ledger, node, world)
			}
		}

		_, containsHash := node.Consensus().FutureBlockQueue()[block.Hash()]
		if containsHash {
			// import future blocks after successful block import
			logger.Audit(node.Id(), "IMPORT_FUTURE", node.Consensus().FutureBlockQueue()[block.Hash()].Hash(), "", node.Time())
			blockToInsert := node.Consensus().FutureBlockQueue()[block.Hash()]
			_, ok := node.Consensus().InsertBlock(blockToInsert, node, ledger, world, node.Consensus().FutureBlockQueueSenderId()[blockToInsert.Hash()], -1)
			if ok {
				delete(node.Consensus().FutureBlockQueue(), block.Hash())
				delete(node.Consensus().FutureBlockQueueSenderId(), blockToInsert.Hash())
			}
		}

		metrics.Timer(interfaces.METRIC_BLOCK_INSERT.String(), ti.Duration(node.Time()-startTime))
		if peerId == node.Id() { // self called with possible uncle block
			// this sets the node time to the right time because of modelled parallelism
			// this happens when between receiving a new block from a peer and finishing its insert an own block is minted
			usedTime := node.Time() - startTime
			if newHead {
				// if a new head was found (= no uncle) then set the event time to the end of inserting the own block
				node.SetTime(evTime + usedTime)
			} else {
				// set back to start time if it was actually an uncle block
				node.SetTime(startTime)
			}
		}
		return newHead, true
	}
}

func (c *SelfishMiningConsensus) NewBlockEvent(node interfaces.INode, block interfaces.IBlock, world interfaces.IWorld, evTime int64) {
	if node.IsOnline() {
		network := node.Network()
		if node.Ledger().Head(node).Hash() == block.ParentHash() {
			// we are selfish, so append block and start mining on top of it but don't publish it yet
			node.Ledger().AppendBlockToCurrent(node, block)
			c.blocksAhead = append(c.blocksAhead, block)
			node.Consensus().MineBlock(node.Ledger(), node, world)
		} else {
			// a block was mined during insert of a received block
			network.BroadcastBlock(block, node, world,
				node.Consensus().BroadcastNewBlockTargets(node, block, true, node.Id())...)
			network.BroadcastBlockHash(block.Hash(), block.Header().Number(), node, world,
				node.Consensus().BroadcastNewBlockTargets(node, block, false, node.Id())...)
			node.Consensus().InsertBlock(block, node, node.Ledger(), world, node.Id(), evTime)
		}
	}
}

func (c *SelfishMiningConsensus) ReceivedBlockEvent(node interfaces.INode, block interfaces.IBlock, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		node.Consensus().MarkBlockSeen(node, block.Hash(), senderId)
		if isRetrieving, ok := node.Consensus().RetrievingHeaders()[block.Hash()]; ok && isRetrieving {
			node.Consensus().RetrievingHeaders()[block.Hash()] = false
		}
		if _, ok := node.Consensus().RetrievingBodies()[block.Hash()]; ok {
			delete(node.Consensus().RetrievingBodies(), block.Hash())
		}

		handleBlockNormally := c.SelfishAttackHandleBlockObserved(block.Header().Number(), block.Hash(), node, world)
		if handleBlockNormally {
			node.Consensus().InsertBlock(block, node, node.Ledger(), world, senderId, -1)
		} else {
			// here we simply trust new blocks for simplicity (beacuse there are no other dishonest nodes)
			// this could also be extended to use an adapted version of InsertBlock
			node.Consensus().WriteBlock(block, node.Ledger(), node, "SIDECHAIN_")
		}
	}
}

func (c *SelfishMiningConsensus) ReceivedBlockHashesEvent(node interfaces.INode, hashes []string, numbers []int, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		minNumber := math.MaxInt64
		minNumberHash := ""
		for i, h := range hashes {
			node.Consensus().MarkBlockSeen(node, h, senderId)
			if !node.Ledger().HasBlock(node, h) {
				isRetrieving, ok := node.Consensus().RetrievingHeaders()[h]
				if !ok || !isRetrieving {
					// we are selfish, handle it but don't depend on the return value
					c.SelfishAttackHandleBlockObserved(numbers[i], h, node, world)
					node.Consensus().RetrievingHeaders()[h] = true
					if numbers[i] < minNumber {
						minNumber = numbers[i]
						minNumberHash = h
					}
				}
			}
		}
		if minNumber != math.MaxInt64 {
			node.Network().RetrieveBlockHeaders(node, world.Nodes()[senderId], world, minNumberHash, len(hashes), false, 0)
		}
	}
}

func (c *SelfishMiningConsensus) ReorgChain(block interfaces.IBlock, ledger interfaces.ILedger, node interfaces.INode, world interfaces.IWorld, auditPrefix string) bool {
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_WRITTEN_REORG, node.Id()), 1)
	ok := ledger.Reorg(node, block, world)
	if !ok {
		logger.Audit(node.Id(), auditPrefix+"CHAIN_REORG_ERROR", block.Hash(), "", node.Time())
		return false
	} else {
		c.blocksAhead = make([]interfaces.IBlock, 0) // delete all blocks we were ahead because the longest chain overtook us
		logger.Audit(node.Id(), auditPrefix+"CHAIN_REORG_BLOCK_WRITTEN", block.Hash(), "", node.Time())
		return true
	}
}

func (c *SelfishMiningConsensus) MineBlock(ledger interfaces.ILedger, node interfaces.INode, world interfaces.IWorld) {
	// just for removing the failing local uncles
	blockTimeStamp, miningTimeDelay := random.TimeBetweenBlocks(world.SimConfig().OverallHashPower(), node.HashPower(), ledger.Head(node).Header().Time(), node.Time())
	miningTime := node.Time() + miningTimeDelay
	newGasLimit := node.Consensus().GetGasLimit(ledger.Head(node).Header(), world)
	txs := make([]interfaces.ITransaction, 0, 50)
	gasAlreadyUsed := 0

	if len(ledger.QueuedTxs()) > 0 {
		localTxs, remoteTxs := ledger.SortedTxByLocalAndRemote(node)
		localsToUse, gasUsedFromLocals := node.Consensus().GetTxsForBlock(gasAlreadyUsed, localTxs, newGasLimit)
		txs = append(txs, localsToUse...)
		gasAlreadyUsed += gasUsedFromLocals
		remotesToUse, gasUsedFromRemotes := node.Consensus().GetTxsForBlock(gasAlreadyUsed, remoteTxs, newGasLimit)
		txs = append(txs, remotesToUse...)
		gasAlreadyUsed += gasUsedFromRemotes
	}

	// fill block with random txs if tx creation propagation is not simulated
	if !world.SimConfig().SimulateTransactionCreation() {
		randomsToUse, gasUsedFromRandoms := events.CreateRandomTransactionsForBlock(gasAlreadyUsed, world, newGasLimit)
		txs = append(txs, randomsToUse...)
		gasAlreadyUsed += gasUsedFromRandoms
	}

	txHash := ""
	for i, tx := range txs {
		txHash += tx.Id()
		if i != len(txs)-1 {
			txHash += ","
		}
	}
	hasher := sha256.New()
	hasher.Write([]byte(txHash))
	txSha256 := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	if len(txs) <= 0 {
		txSha256 = ""
	}

	uncles := make([]interfaces.IBlockHeader, 0, 2)
	uncleHashes := make([]string, 0, 2)
	if len(ledger.PossibleUncles()) > 0 {
		localUncles := make([]interfaces.IBlockHeader, 0, 2)
		remoteUncles := make([]interfaces.IBlockHeader, 0, 2)
		// divide into local and remote uncles
		for _, uncle := range ledger.PossibleUncles() {
			if uncle.MinerId() == node.Id() {
				localUncles = append(localUncles, uncle)
			} else {
				remoteUncles = append(remoteUncles, uncle)
			}
		}
		/*// add valid local uncles
		usedLocalUncles, usedLocalUncleHashes := getSelfishUncles(localUncles, world, ledger, node, 0)
		// sort because of determinism
		sort.Slice(usedLocalUncles, func(i, j int) bool {
			return usedLocalUncles[i].Number() < usedLocalUncles[j].Number()
		})
		uncles = append(uncles, usedLocalUncles...)
		uncleHashes = append(uncleHashes, usedLocalUncleHashes...)*/

		// add valid remote uncles if not already 2 found
		if len(uncles) < 2 {
			usedRemoteUncles, usedRemoteUncleHashes := getSelfishUncles(remoteUncles, world, ledger, node, len(uncles))
			// sort because of determinism
			sort.Slice(usedRemoteUncles, func(i, j int) bool {
				return usedRemoteUncles[i].Number() < usedRemoteUncles[j].Number()
			})
			uncles = append(uncles, usedRemoteUncles...)
			uncleHashes = append(uncleHashes, usedRemoteUncleHashes...)
		}
	}
	uncleHash := strings.Join(uncleHashes, ",")
	blockSize := world.SimConfig().Sizes()["header"] + world.SimConfig().Sizes()["tx"]*len(txs) + world.SimConfig().Sizes()["header"]*len(uncles)
	header := ledg.NewBlockHeader(world.NewBlockHash(), txSha256, uncleHash, ledger.Head(node).Hash(), node.Id(), node.Consensus().CalcDifficulty(ledger.Head(node).Header(), blockTimeStamp, world), gasAlreadyUsed, newGasLimit, blockTimeStamp, ledger.Head(node).Header().Number()+1, blockSize, true)
	body := ledg.NewBlockBody(header.Hash(), txs, uncles, true, len(txs))
	block := ledg.NewBlock(header, body, ledger.Head(node).TotalDifficulty()+header.Difficulty())
	ev := events.NewNewBlockEvent(event.NewEvent(miningTime, node.Id(), interfaces.NEW_BLOCK_EVENT), block, node.Time())
	world.Queue().Add(ev)
}

func getSelfishUncles(possibleUncles []interfaces.IBlockHeader, world interfaces.IWorld, ledger interfaces.ILedger, node interfaces.INode, alreadyUsedUncles int) (uncles []interfaces.IBlockHeader, uncleHashes []string) {
	uncles = make([]interfaces.IBlockHeader, 0, 2-alreadyUsedUncles)
	uncleHashes = make([]string, 0, 2-alreadyUsedUncles)
	for _, uncle := range possibleUncles {
		if uncle.Number()+int(world.SimConfig().MaxUncleDist()) < ledger.Length(node) {
			delete(ledger.PossibleUncles(), uncle.Hash())
			continue
		}
		if _, exists := ledger.Uncles()[uncle.Hash()]; exists {
			delete(ledger.PossibleUncles(), uncle.Hash())
			continue
		}
		uncleHashes = append(uncleHashes, uncle.Hash())
		uncles = append(uncles, uncle)
		if len(uncleHashes) == 2-alreadyUsedUncles {
			break
		}
	}
	return
}

func (c *SelfishMiningConsensus) SelfishAttackHandleBlockObserved(observedBlockNumber int, observedBlockHash string, node interfaces.INode, world interfaces.IWorld) (handleBlockNormally bool) {

	alreadyHandledNum, okNum := c.blockHandledNum[observedBlockNumber]
	if alreadyHandledNum && okNum {
		return true
	}
	c.blockHandledNum[observedBlockNumber] = true

	if len(c.blocksAhead) <= 0 {
		// no selfish mining attack happening, abort handling
		return true
	}

	currentHead := node.Ledger().Head(node)
	if currentHead.Header().Number() == observedBlockNumber || currentHead.Header().Number() == observedBlockNumber+1 {
		// we were either 1 block ahead, now off to a block race
		// or we were 2 blocks ahead, now only one left -> publish all blocks
		for i := 0; i < len(c.blocksAhead); i++ {
			blockToPublish := c.blocksAhead[i]
			network := node.Network()
			network.BroadcastBlock(blockToPublish, node, world, c.SelfishAttackBroadcastBlockTargets(node, blockToPublish)...)
		}
		c.blocksAhead = make([]interfaces.IBlock, 0)
		return true
	}

	if currentHead.Header().Number() >= observedBlockNumber+2 {
		// we were at least 3 blocks ahead, now one less -> publish first unpublished block
		blockToPublish := c.blocksAhead[0]
		c.blocksAhead = c.blocksAhead[1:]
		network := node.Network()
		network.BroadcastBlock(blockToPublish, node, world, c.SelfishAttackBroadcastBlockTargets(node, blockToPublish)...)
		return true
	}

	for i := 0; i < len(c.blocksAhead); i++ {
		blockToPublish := c.blocksAhead[i]
		network := node.Network()
		network.BroadcastBlock(blockToPublish, node, world, c.SelfishAttackBroadcastBlockTargets(node, blockToPublish)...)
	}
	c.blocksAhead = make([]interfaces.IBlock, 0)
	return true
}

func (c *SelfishMiningConsensus) SelfishAttackBroadcastBlockTargets(node interfaces.INode, block interfaces.IBlock) (targets []interfaces.INode) {
	selectedPeers := make([]interfaces.INode, 0, len(node.Peers()))
	for _, peer := range node.Peers() {
		if _, ok := node.Consensus().BlockSeen()[block.Hash()][peer.Id()]; !ok {
			selectedPeers = append(selectedPeers, peer)
		}
	}
	// send whole block to all peers immediately
	targets = selectedPeers
	for _, target := range targets {
		node.Consensus().MarkBlockSeen(node, block.Hash(), target.Id())
	}
	return
}
