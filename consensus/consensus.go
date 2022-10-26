package consensus

import (
	"crypto/sha256"
	"encoding/base64"
	"ethattacksim/event"
	"ethattacksim/event/events"
	"ethattacksim/interfaces"
	ledg "ethattacksim/ledger"
	"ethattacksim/util/helper"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"ethattacksim/util/random"
	"math"
	"sort"
	"strings"
	ti "time"
)

type Consensus struct {
	blockSeen                map[string]map[string]bool         // peers
	txSeen                   map[string]map[string]bool         // peers
	retrievingHeaders        map[string]bool                    // indicates if header for block hash is retrieving
	retrievingBodies         map[string]interfaces.IBlockHeader // indicates if body is retrieving and caches header
	futureBlockQueue         map[string]interfaces.IBlock       // parent hash to block
	futureBlockQueueSenderId map[string]string                  // block hash to sender id, only for dropping peer
}

func NewConsensus() interfaces.IConsensus {
	return &Consensus{blockSeen: make(map[string]map[string]bool, 1000), txSeen: make(map[string]map[string]bool, 1000), retrievingHeaders: make(map[string]bool, 1000), retrievingBodies: make(map[string]interfaces.IBlockHeader, 1000), futureBlockQueue: make(map[string]interfaces.IBlock, 20), futureBlockQueueSenderId: make(map[string]string, 20)}
}

func (c *Consensus) BlockSeen() map[string]map[string]bool {
	return c.blockSeen
}

func (c *Consensus) TxSeen() map[string]map[string]bool {
	return c.txSeen
}

func (c *Consensus) RetrievingHeaders() map[string]bool {
	return c.retrievingHeaders
}

func (c *Consensus) RetrievingBodies() map[string]interfaces.IBlockHeader {
	return c.retrievingBodies
}

func (c *Consensus) FutureBlockQueue() map[string]interfaces.IBlock {
	return c.futureBlockQueue
}

func (c *Consensus) FutureBlockQueueSenderId() map[string]string {
	return c.futureBlockQueueSenderId
}

func (c *Consensus) ReceivedBlockEvent(node interfaces.INode, block interfaces.IBlock, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		node.Consensus().MarkBlockSeen(node, block.Hash(), senderId)
		if isRetrieving, ok := node.Consensus().RetrievingHeaders()[block.Hash()]; ok && isRetrieving {
			node.Consensus().RetrievingHeaders()[block.Hash()] = false
		}
		if _, ok := node.Consensus().RetrievingBodies()[block.Hash()]; ok {
			delete(node.Consensus().RetrievingBodies(), block.Hash())
		}
		node.Consensus().InsertBlock(block, node, node.Ledger(), world, senderId, -1)
	}
}

func (c *Consensus) NewBlockEvent(node interfaces.INode, block interfaces.IBlock, world interfaces.IWorld, evTime int64) {
	if node.IsOnline() {
		network := node.Network()
		if node.Ledger().Head(node).Hash() == block.ParentHash() {
			node.Ledger().AppendBlockToCurrent(node, block)
			network.BroadcastBlock(block, node, world,
				node.Consensus().BroadcastNewBlockTargets(node, block, true, node.Id())...)
			network.BroadcastBlockHash(block.Hash(), block.Header().Number(), node, world,
				node.Consensus().BroadcastNewBlockTargets(node, block, false, node.Id())...)
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

func (c *Consensus) ReceivedBlockHashesEvent(node interfaces.INode, hashes []string, numbers []int, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		minNumber := math.MaxInt64
		minNumberHash := ""
		for i, h := range hashes {
			node.Consensus().MarkBlockSeen(node, h, senderId)
			if !node.Ledger().HasBlock(node, h) {
				isRetrieving, ok := node.Consensus().RetrievingHeaders()[h]
				if !ok || !isRetrieving {
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

func (c *Consensus) RetrieveBlockHeadersEvent(node interfaces.INode, originBlockHash string, num int, reverse bool, skip int, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		headers := node.Consensus().RetrieveHeaders(node, node.Ledger(), originBlockHash, num, reverse, skip)
		if len(headers) != 0 {
			node.Network().SendBlockHeaders(node, world.Nodes()[senderId], world, headers)
		}
	}
}

func (c *Consensus) ReceivedTxsEvent(node interfaces.INode, txs []interfaces.ITransaction, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		for _, tx := range txs {
			node.Consensus().MarkTxSeen(node, tx.Id(), senderId)
			if !node.Ledger().KnowsQueuedTx(node, tx.Id()) {
				ok := node.Consensus().VerifyTx(tx, node)
				if ok {
					// add to queue and broadcast if valid
					node.Ledger().AddTxsToQueue(node, tx)
					broadcastPropagateTargets := node.Consensus().BroadcastTxTargets(node, tx, true, node.Id())
					node.Network().BroadcastTxs([]interfaces.ITransaction{tx}, node, world, broadcastPropagateTargets...)
					broadcastOtherTargets := node.Consensus().BroadcastTxTargets(node, tx, false, node.Id())
					node.Network().BroadcastTxHashes([]string{tx.Id()}, node, world, broadcastOtherTargets...)
				}
			}
		}
	}
}

func (c *Consensus) ReceivedTxHashesEvent(node interfaces.INode, txHashes []string, senderId string, world interfaces.IWorld) {
	toRetrieve := make([]string, 0, len(txHashes))
	if node.IsOnline() {
		for _, txHash := range txHashes {
			node.Consensus().MarkTxSeen(node, txHash, senderId)
			if !node.Ledger().KnowsQueuedTx(node, txHash) {
				toRetrieve = append(toRetrieve, txHash)
			}
		}
		node.Network().RetrieveTxs(toRetrieve, node, world.Nodes()[senderId], world)
	}
}

func (c *Consensus) RetrieveTxsEvent(node interfaces.INode, txHashes []string, senderId string, world interfaces.IWorld) {
	ret := make([]interfaces.ITransaction, 0, len(txHashes))
	if node.IsOnline() {
		for _, txHash := range txHashes {
			if node.Ledger().KnowsQueuedTx(node, txHash) {
				ret = append(ret, node.Ledger().GetTx(node, txHash))
				node.Consensus().MarkTxSeen(node, txHash, senderId)
			}
		}
		node.Network().BroadcastTxs(ret, node, world, world.Nodes()[senderId])
	}
}

func (c *Consensus) ReceivedBlockHeadersEvent(node interfaces.INode, headers []interfaces.IBlockHeader, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		retrieveHashes := make([]string, 0, len(headers))
		blocksToImport := make([]interfaces.IBlock, 0)
		for _, header := range headers {
			if isRetrieving, ok := node.Consensus().RetrievingHeaders()[header.Hash()]; ok && isRetrieving && !node.Ledger().HasBlock(node, header.Hash()) {
				node.Consensus().RetrievingHeaders()[header.Hash()] = false
				if header.TxHash() == "" && header.UncleHash() == "" {
					// block is empty, import now without retrieving body
					body := ledg.NewBlockBody(header.Hash(), make([]interfaces.ITransaction, 0), make([]interfaces.IBlockHeader, 0), true, 0)
					blocksToImport = append(blocksToImport, ledg.NewBlock(header, body, -1))
				} else {
					retrieveHashes = append(retrieveHashes, header.Hash())
					node.Consensus().RetrievingBodies()[header.Hash()] = header
				}
			}
		}
		if len(retrieveHashes) > 0 {
			node.Network().RetrieveBlockBodies(node, world.Nodes()[senderId], world, retrieveHashes)
		}
		for _, b := range blocksToImport {
			node.Consensus().InsertBlock(b, node, node.Ledger(), world, senderId, -1)
		}
	}
}

func (c *Consensus) RetrieveBlockBodiesEvent(node interfaces.INode, hashes []string, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		bodies := node.Consensus().RetrieveBodies(node, node.Ledger(), hashes)
		for _, body := range bodies {
			node.Consensus().MarkBlockSeen(node, body.BlockHash(), senderId)
		}
		if len(bodies) != 0 {
			node.Network().SendBlockBodies(node, world.Nodes()[senderId], world, bodies)
		}
	}
}

func (c *Consensus) ReceivedBlockBodiesEvent(node interfaces.INode, bodies []interfaces.IBlockBody, senderId string, world interfaces.IWorld) {
	if node.IsOnline() {
		for _, body := range bodies {
			if header, ok := node.Consensus().RetrievingBodies()[body.BlockHash()]; ok {
				delete(node.Consensus().RetrievingBodies(), body.BlockHash())
				// totalDifficulty will be set later on when verifying header
				node.Consensus().InsertBlock(ledg.NewBlock(header, body, -1), node, node.Ledger(), world, senderId, -1)
			}
		}
	}
}

func (c *Consensus) InsertBlock(block interfaces.IBlock, node interfaces.INode, ledger interfaces.ILedger, world interfaces.IWorld, peerId string, evTime int64) (newHead bool, ok bool) {
	startTime := node.Time()
	switch {
	case block.Header().Number() > ledger.Length(node): // ledger.Length() == headBlock number + 1
		node.Consensus().FutureBlockQueue()[block.ParentHash()] = block
		node.Consensus().FutureBlockQueueSenderId()[block.Hash()] = peerId
		logger.Audit(node.Id(), "FUTURE_BLOCK", block.Hash(), "", node.Time())
		return false, false
	case block.Header().Number()+int(world.SimConfig().MaxUncleDist()) < ledger.Length(node)-1:
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

func (c *Consensus) InsertToSidechain(blocks []interfaces.IBlock, headerErrors []error, node interfaces.INode, ledger interfaces.ILedger, world interfaces.IWorld) (newHead bool, ok bool) {
	localTd := node.Consensus().TotalDifficulty(node, ledger.Head(node).Hash(), ledger)
	externalTd := 0
	var lastBlock interfaces.IBlock
	for i, block := range blocks {
		err := headerErrors[i]
		if err == nil {
			err := node.Consensus().VerifyBody(block, ledger, world, node)
			if err == interfaces.ErrPrunedAncestor {
				lastBlock = block
				logger.Audit(node.Id(), "PRUNED_ANCESTOR", block.Hash(), "", node.Time())
				if ledger.HasBlock(node, block.Hash()) {
					externalTd = node.Consensus().TotalDifficulty(node, block.Hash(), ledger)
				} else {
					externalTd = node.Consensus().TotalDifficulty(node, block.Header().ParentHash(), ledger) + block.Header().Difficulty()
					node.Consensus().WriteBlock(block, ledger, node, "SIDECHAIN_")
				}
			} else {
				break
			}
		}
	}
	if lastBlock == nil {
		logger.Audit(node.Id(), "SIDECHAIN_NO_LAST_BLOCK", "", "", node.Time())
		return false, false
	}
	reorg := node.Consensus().CheckReorg(localTd, externalTd, lastBlock, ledger.Head(node), node)
	if !reorg {
		logger.Audit(node.Id(), "SIDECHAIN_TD_LOW", "", "", node.Time())
		return false, true
	}

	// collect blocks to import
	toImport := ledger.GetBlock(node, lastBlock.Hash())
	blocksToImport := make([]interfaces.IBlock, 0, len(blocks))
	for toImport != nil && !ledger.CurrentHasBlock(node, toImport.Hash()) {
		blocksToImport = append(blocksToImport, toImport)
		toImport = ledger.GetBlock(node, toImport.ParentHash())
	}
	ok = len(blocksToImport) > 0

	if ok {
		// verify all blocks to import (to prevent having to roll back in case of state error)
		for _, b := range blocksToImport {
			valid := node.Consensus().VerifyState(b, node, world.SimConfig().CheckPastTxWhenVerifyingState())
			if !valid {
				return false, false
			}
		}

		// reorg by setting head to first block not in current chain (= the last in blocksToImport)
		b := blocksToImport[len(blocksToImport)-1]
		ok = node.Consensus().ReorgChain(b, ledger, node, world, "SIDE")
		blocksToImport = blocksToImport[:len(blocksToImport)-1]
	}
	if ok {
		// append remaining blocks if reorg worked (reverse order!)
		for i := len(blocksToImport) - 1; i >= 0; i-- {
			b := blocksToImport[i]
			node.Consensus().AppendBlock(b, ledger, node, "SIDECHAIN_")
		}
	}
	if !ok {
		return false, false
	} else {
		return true, true
	}
}

func (c *Consensus) InsertToChain(blocks []interfaces.IBlock, node interfaces.INode, ledger interfaces.ILedger, world interfaces.IWorld) (newHead bool, ok bool) {
	headerErrors := node.Consensus().VerifyHeaders(blocks, ledger, world, node)

	for i, block := range blocks {
		err := headerErrors[i]
		if err == nil {
			err = node.Consensus().VerifyBody(block, ledger, world, node)
		}

		switch {
		case err == interfaces.ErrKnownBlock:
			continue
		case err == interfaces.ErrPrunedAncestor:
			newHead, ok := node.Consensus().InsertToSidechain(blocks[i:], headerErrors[i:], node, ledger, world)
			return newHead, ok
		case err == interfaces.ErrUnknownAncestor:
			return false, false
		// not modelled
		//case err == interfaces.ErrFutureBlock:
		//node.Consensus().futureBlockQueue[block.ParentHash()] = block
		case err != nil:
			logger.Audit(node.Id(), "UNKNOWN_BLOCK_ERROR", block.Hash(), "", node.Time())
			return false, false
		default:
			valid := node.Consensus().VerifyState(block, node, world.SimConfig().CheckPastTxWhenVerifyingState())
			if !valid {
				return false, false
			}
			switch {
			case block.ParentHash() == ledger.Head(node).Hash():
				// simply appending to head
				node.Consensus().AppendBlock(block, ledger, node, "")
			case ledger.CurrentHasBlock(node, block.ParentHash()):
				localTd := node.Consensus().TotalDifficulty(node, ledger.Head(node).Hash(), ledger)
				externalTd := node.Consensus().TotalDifficulty(node, block.ParentHash(), ledger) + block.Header().Difficulty() // already verified here
				reorg := node.Consensus().CheckReorg(localTd, externalTd, block, ledger.Head(node), node)
				if reorg {
					ok := node.Consensus().ReorgChain(block, ledger, node, world, "")
					if !ok {
						return false, false
					}
				} else {
					node.Consensus().WriteBlock(block, ledger, node, "")
					return false, true
				}
			default:
				logger.Audit(node.Id(), "UNKNOWN_BLOCK_STATUS", block.Hash(), "", node.Time())
				return false, false
			}
		}
	}
	return true, true
}

func (c *Consensus) CheckReorg(localTd int, externalTd int, block interfaces.IBlock, currentHead interfaces.IBlock, node interfaces.INode) bool {
	if localTd > externalTd {
		return false
	} else if localTd == externalTd {
		if block.Header().Number() < currentHead.Header().Number() {
			return true
		} else if block.Header().Number() == currentHead.Header().Number() {
			// see core/blockchain#writeBlockWithState)
			currentPreserve, blockPreserve := currentHead.Header().MinerId() == node.Id(), block.Header().MinerId() == node.Id()
			return !currentPreserve && (blockPreserve || random.Uniform() < 0.5)
		} else {
			return false
		}
	} else {
		return true
	}
}

func (c *Consensus) WriteBlock(block interfaces.IBlock, ledger interfaces.ILedger, node interfaces.INode, auditPrefix string) {
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_WRITTEN, node.Id()), 1)
	logger.Audit(node.Id(), auditPrefix+"BLOCK_WRITTEN", block.Hash(), "", node.Time())
	ledger.WriteBlock(node, block, false)
}

func (c *Consensus) AppendBlock(block interfaces.IBlock, ledger interfaces.ILedger, node interfaces.INode, auditPrefix string) {
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_APPENDED, node.Id()), 1)
	logger.Audit(node.Id(), auditPrefix+"BLOCK_APPENDED", block.Hash(), "", node.Time())
	ledger.AppendBlockToCurrent(node, block)
}

func (c *Consensus) ReorgChain(block interfaces.IBlock, ledger interfaces.ILedger, node interfaces.INode, world interfaces.IWorld, auditPrefix string) bool {
	metrics.Counter(metrics.NameFormat(interfaces.METRIC_BLOCK_WRITTEN_REORG, node.Id()), 1)
	ok := ledger.Reorg(node, block, world)
	if !ok {
		logger.Audit(node.Id(), auditPrefix+"CHAIN_REORG_ERROR", block.Hash(), "", node.Time())
		return false
	} else {
		logger.Audit(node.Id(), auditPrefix+"CHAIN_REORG_BLOCK_WRITTEN", block.Hash(), "", node.Time())
		return true
	}
}

func (c *Consensus) RetrieveHeaders(node interfaces.INode, ledger interfaces.ILedger, originBlockHash string, num int, reverse bool, skip int) []interfaces.IBlockHeader {
	headers := make([]interfaces.IBlockHeader, 0, num)
	if !ledger.HasBlock(node, originBlockHash) {
		return headers
	}
	currentHead := ledger.GetBlock(node, originBlockHash).Header()
	headers = append(headers, currentHead)
	for i := 1; i < num; i++ { // start at 1 because origin already found
		if reverse {
			for i := 0; i < skip+1; i++ {
				if !ledger.HasBlock(node, currentHead.ParentHash()) {
					currentHead = nil
					break
				}
				currentHead = ledger.GetBlock(node, currentHead.ParentHash()).Header()
			}
			if currentHead == nil {
				break
			}
			headers = append([]interfaces.IBlockHeader{currentHead}, headers...)
		} else {
			b := ledger.CurrentGetBlockByNumber(node, currentHead.Number()+1+skip)
			if b == nil {
				break
			}
			currentHead = b.Header()
			headers = append(headers, currentHead)
		}
	}
	return headers
}

func (c *Consensus) RetrieveBodies(node interfaces.INode, ledger interfaces.ILedger, hashes []string) []interfaces.IBlockBody {
	bodies := make([]interfaces.IBlockBody, 0, len(hashes))
	for _, hash := range hashes {
		if ledger.HasBlock(node, hash) {
			bodies = append(bodies, ledger.GetBlock(node, hash).Body())
		}
	}
	return bodies
}

// still returns timeConsumed because verifyHeaders() needs it
func (c *Consensus) VerifyHeader(block interfaces.IBlock, ledger interfaces.ILedger, world interfaces.IWorld, node interfaces.INode, isUncle bool) (timeconsumed int64, err error) {
	timeToAdd := random.BaseHeaderVerification(node.HashPower(), node.CpuPower())
	node.IncrementTime(timeToAdd)
	if !isUncle {
		if ledger.HasBlock(node, block.Header().Hash()) {
			return timeToAdd, nil
		}
	}
	if !ledger.HasBlock(node, block.ParentHash()) {
		return timeToAdd, interfaces.ErrUnknownAncestor
	}
	// not modelled
	/*if !isUncle {
		if block.Header().Time() > (time + 15000000000) {
			return interfaces.ErrFutureBlock
		}
	}*/
	// subsequent block has to have a timestamp of at least one second more than parent
	if block.Header().Time() < ledger.GetBlock(node, block.ParentHash()).Header().Time()+1 {
		return timeToAdd, interfaces.ErrOlderBlock
	}
	diff := node.Consensus().CalcDifficulty(ledger.GetBlock(node, block.ParentHash()).Header(), block.Header().Time(), world)
	if diff != block.Header().Difficulty() {
		return timeToAdd, interfaces.ErrInvalidHeader
	} else {
		if block.TotalDifficulty() == -1 {
			// set total difficulty to block if not set and already verified
			block.SetTotalDifficulty(ledger.GetBlock(node, block.ParentHash()).TotalDifficulty() + block.Header().Difficulty())
		}
	}
	// check gas limit/usage
	if block.Header().GasUsed() > block.Header().GasLimit() {
		return timeToAdd, interfaces.ErrInvalidHeader
	}
	maxGasAdaption := ledger.GetBlock(node, block.ParentHash()).Header().GasLimit() / 1024
	gasDiff := ledger.GetBlock(node, block.ParentHash()).Header().GasLimit() - block.Header().GasLimit()
	if gasDiff < 0 {
		gasDiff = gasDiff * -1
	}
	if gasDiff > maxGasAdaption {
		return timeToAdd, interfaces.ErrInvalidHeader
	}

	if !block.Header().IsValid() {
		return timeToAdd, interfaces.ErrInvalidHeader
	}
	return timeToAdd, nil
}

// verifies headers in parallel
func (c *Consensus) VerifyHeaders(blocks []interfaces.IBlock, ledger interfaces.ILedger, world interfaces.IWorld, node interfaces.INode) (errors []error) {
	errors = make([]error, len(blocks), len(blocks))
	startNodeTime := node.Time()
	var timeToAdd int64 = 0
	for i, block := range blocks {
		addTime, err := node.Consensus().VerifyHeader(block, ledger, world, node, false)
		errors[i] = err
		if addTime > timeToAdd {
			timeToAdd = addTime // just replace time with highest observed value as headers are validated in parallel
		}
	}
	node.SetTime(startNodeTime + timeToAdd)
	return errors
}

func (c *Consensus) VerifyBody(block interfaces.IBlock, ledger interfaces.ILedger, world interfaces.IWorld, node interfaces.INode) (err error) {
	node.IncrementTime(random.BaseBodyVerification(node.HashPower(), node.CpuPower()))
	// check if known
	if ledger.HasBlock(node, block.Body().BlockHash()) {
		return interfaces.ErrKnownBlock
	}
	// verify uncles
	for _, u := range block.Body().Uncles() {
		uncleParent := ledger.GetBlock(node, u.ParentHash())
		if uncleParent == nil || block.ParentHash() == u.ParentHash() {
			return interfaces.ErrDanglingUncle
		}

		// check if an uncle is an ancestor (or in uncles of an ancestor) of the block to validate
		ancestorHashes := make(map[string]bool, 0)
		uncleHashes := make(map[string]bool, 0)
		parent := ledger.GetBlock(node, block.ParentHash())
		for parent != nil {
			ancestorHashes[parent.Hash()] = true
			for _, u := range parent.Body().Uncles() {
				uncleHashes[u.Hash()] = true
			}
			parent = ledger.GetBlock(node, parent.ParentHash())
		}
		if _, exists := ancestorHashes[u.Hash()]; exists {
			return interfaces.ErrUncleIsAncestor
		}
		if _, exists := uncleHashes[u.Hash()]; exists {
			return interfaces.ErrUncleIsAncestor
		}

		uncleBlock := ledg.NewBlock(u, ledg.NewBlockBody(u.Hash(), make([]interfaces.ITransaction, 0), make([]interfaces.IBlockHeader, 0), true, 0), -1)
		_, err := node.Consensus().VerifyHeader(uncleBlock, ledger, world, node, true)
		if err != nil {
			return err
		}
	}
	// check uncle hash
	// check tx hash

	if !block.Body().IsValid() {
		return interfaces.ErrInvalidBody
	}
	if !ledger.CurrentHasBlock(node, block.ParentHash()) {
		if !ledger.HasBlock(node, block.ParentHash()) {
			return interfaces.ErrUnknownAncestor
		}
		return interfaces.ErrPrunedAncestor
	}
	return nil
}

func (c *Consensus) VerifyState(block interfaces.IBlock, node interfaces.INode, checkPastTx bool) (ok bool) {
	// compute and verify txs/state sequentially
	invalidTxFound := false

	ancestorTxs := make(map[string]bool, 0)

	if checkPastTx {
		parent := node.Ledger().GetBlock(node, block.ParentHash())
		for parent != nil {
			for _, t := range parent.Body().Transactions() {
				ancestorTxs[t.Id()] = true
			}
			parent = node.Ledger().GetBlock(node, parent.ParentHash())
		}
	}

	for _, tx := range block.Body().Transactions() {
		invalidTxFound = !node.Consensus().VerifyTx(tx, node)
		node.IncrementTime(random.TxStateComputation(tx.GasUsed(), node.CpuPower(), tx.SpecialTxStateComputation()))

		if checkPastTx && !invalidTxFound {
			if _, exists := ancestorTxs[tx.Id()]; exists {
				invalidTxFound = true
			}
		}

		if invalidTxFound {
			break
		}
	}
	return !invalidTxFound
}

// basic TX verification, no state computation
func (c *Consensus) VerifyTx(tx interfaces.ITransaction, node interfaces.INode) (ok bool) {
	node.IncrementTime(random.BaseTxVerification(node.HashPower(), node.CpuPower()))
	// check tx size
	// check sender nonce
	// check signed
	// check gas price
	// check gas > currentMaxGas
	// check sender balance high enough
	// check tx gas higher than min gas (21000)
	return tx.IsValid()
}

func (c *Consensus) CalcDifficulty(parentHeader interfaces.IBlockHeader, time int64, world interfaces.IWorld) int {
	// simply giving higher difficulty to blocks that were created quicker
	// blocks with uncles have also higher difficulty
	/*diffS := time - parentHeader.Time()
	if len(parentHeader.UncleHash()) > 0 {
		fmt.Printf("diffS %v, d %v\n", diffS, parentHeader.Difficulty() + 30 - int(diffS))
		return parentHeader.Difficulty() + 30 - int(diffS)
	} else {
		fmt.Printf("diffS %v, d %v\n", diffS, parentHeader.Difficulty() + 20 - int(diffS))
		return parentHeader.Difficulty() + 20 - int(diffS)
	}*/

	// Explanation:
	// y is set to 8 by choice to keep ints down & keep algorithm simple (for normal this depends on the difficulty of
	// the parent, but as this simulator does not have a mining algorithm depending on the difficulty this can be neglected)
	// difficulty rises by y if time between blocks was <= 9 seconds
	// difficulty stays the same if time between blocks was 9 < x <= 18 seconds
	// difficulty falls by y for every 9 seconds the time between blocks was larger than 18 seconds with a max of 99 * y seconds
	// having uncles adds an extra difficulty of y
	// difficulty cannot be below y

	parentTime := parentHeader.Time()
	x := time - parentTime
	x = x / 9
	if len(parentHeader.UncleHash()) > 0 {
		x = 2 - x
	} else {
		x = 1 - x
	}
	if x < -99 {
		x = -99
	}

	y := int64(8)
	// instead of
	// int64(parentHeader.Difficulty() / 2048)

	x = x * y
	x = int64(parentHeader.Difficulty()) + x

	if x < 8 {
		x = 8
	}
	/* // instead of
	if x < 131072 {
		x = 131072
	}*/

	/*// leaving bomb delay out of simulation
	bombDelayFromParent := world.SimConfig().BombDelay() - 1
	var fakeBlockNumber int64 = 0
	if int64(parentHeader.Number()) >= int64(bombDelayFromParent) {
		fakeBlockNumber = int64(parentHeader.Number()) - int64(bombDelayFromParent)
	}
	periodCount := fakeBlockNumber
	periodCount = periodCount / 100000
	if periodCount > 1 {
		y = periodCount - 2
		y = int64(math.Pow(2, float64(y)))
		y = 2 ^ y
		x = x + y
	}*/
	return int(x)
}

func (c *Consensus) TotalDifficulty(node interfaces.INode, hash string, ledger interfaces.ILedger) int {
	return ledger.GetBlock(node, hash).TotalDifficulty()
}

func (c *Consensus) BroadcastNewBlockTargets(node interfaces.INode, block interfaces.IBlock, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	numPeers := int(math.Sqrt(float64(len(node.Peers()))))
	selectedPeers := make([]interfaces.INode, 0, len(node.Peers()))
	for _, peer := range node.Peers() {
		if _, ok := node.Consensus().BlockSeen()[block.Hash()][peer.Id()]; !ok {
			if !helper.ContainsString(excludeIds, peer.Id()) {
				selectedPeers = append(selectedPeers, peer)
			}
		}
	}
	if propagate {
		targets = selectedPeers[:int(math.Min(float64(numPeers), float64(len(selectedPeers))))]
	} else {
		targets = selectedPeers
	}
	for _, target := range targets {
		node.Consensus().MarkBlockSeen(node, block.Hash(), target.Id())
	}
	return
}

func (c *Consensus) BroadcastReceivedBlockTargets(node interfaces.INode, block interfaces.IBlock, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	targets = node.Consensus().BroadcastNewBlockTargets(node, block, propagate, excludeIds...)
	return
}

func (c *Consensus) BroadcastTxTargets(node interfaces.INode, tx interfaces.ITransaction, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	numPeers := int(math.Sqrt(float64(len(node.Peers()))))
	selectedPeers := make([]interfaces.INode, 0, len(node.Peers()))
	for _, peer := range node.Peers() {
		if _, ok := node.Consensus().TxSeen()[tx.Id()][peer.Id()]; !ok {
			if !helper.ContainsString(excludeIds, peer.Id()) {
				selectedPeers = append(selectedPeers, peer)
			}
		}
	}
	if propagate {
		targets = selectedPeers[:int(math.Min(float64(numPeers), float64(len(selectedPeers))))]
	} else {
		targets = selectedPeers
	}
	for _, target := range targets {
		node.Consensus().MarkTxSeen(node, tx.Id(), target.Id())
	}
	return
}

func (c *Consensus) MineBlock(ledger interfaces.ILedger, node interfaces.INode, world interfaces.IWorld) {
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
		// add valid local uncles
		usedLocalUncles, usedLocalUncleHashes := getUncles(localUncles, world, ledger, node, 0)
		// sort because of determinism
		sort.Slice(usedLocalUncles, func(i, j int) bool {
			return usedLocalUncles[i].Number() < usedLocalUncles[j].Number()
		})
		uncles = append(uncles, usedLocalUncles...)
		uncleHashes = append(uncleHashes, usedLocalUncleHashes...)

		// add valid remote uncles if not already 2 found
		if len(uncles) < 2 {
			usedRemoteUncles, usedRemoteUncleHashes := getUncles(remoteUncles, world, ledger, node, len(uncles))
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

func getUncles(possibleUncles []interfaces.IBlockHeader, world interfaces.IWorld, ledger interfaces.ILedger, node interfaces.INode, alreadyUsedUncles int) (uncles []interfaces.IBlockHeader, uncleHashes []string) {
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

func (c *Consensus) GetTxsForBlock(gasUsed int, txs []interfaces.ITransaction, gasLimit int) (transactions []interfaces.ITransaction, gasAmount int) {
	transactions = make([]interfaces.ITransaction, 0)
	gasAmount = 0
	for _, tx := range txs {
		if gasUsed+gasAmount+tx.GasUsed() > gasLimit {
			break
		}
		transactions = append(transactions, tx)
		gasAmount += tx.GasUsed()
	}
	return transactions, gasAmount
}

func (c *Consensus) GetGasLimit(currentHead interfaces.IBlockHeader, world interfaces.IWorld) (gasLimit int) {
	rand := random.Uniform()
	doIncreaseGas := rand < 0.1                  // some nodes may try to increase the limit
	doDecreaseGas := rand > 0.9                  // some nodes may try to decrease the limit
	maxAdaption := currentHead.GasLimit() / 1024 // maximum allowed change of gas limit
	if doIncreaseGas {
		gasLimit = currentHead.GasLimit() + maxAdaption
	} else if doDecreaseGas {
		gasLimit = currentHead.GasLimit() - maxAdaption
	} else {
		// try keep initial
		if currentHead.GasLimit() > world.SimConfig().Limits()["initialGasLimit"] {
			if currentHead.GasLimit()-maxAdaption > world.SimConfig().Limits()["initialGasLimit"] {
				gasLimit = currentHead.GasLimit() - maxAdaption
			} else {
				gasLimit = world.SimConfig().Limits()["initialGasLimit"]
			}
		} else {
			if currentHead.GasLimit()+maxAdaption < world.SimConfig().Limits()["initialGasLimit"] {
				gasLimit = currentHead.GasLimit() + maxAdaption
			} else {
				gasLimit = world.SimConfig().Limits()["initialGasLimit"]
			}
		}
	}
	return
}

func (c *Consensus) MarkBlockSeen(node interfaces.INode, hash string, peerId string) {
	if node.Consensus().BlockSeen()[hash] == nil {
		node.Consensus().BlockSeen()[hash] = make(map[string]bool)
	}
	// hardcoded max blocks monitored
	if len(node.Consensus().BlockSeen()) >= 1024 {
		// this adds indeterminism as range over maps is defined but not guaranteed indeterministic
		for k := range node.Consensus().BlockSeen() {
			if k != hash {
				delete(node.Consensus().BlockSeen(), k)
				return
			}
		}
	}
	node.Consensus().BlockSeen()[hash][peerId] = true
}

func (c *Consensus) MarkTxSeen(node interfaces.INode, hash string, peerId string) {
	if node.Consensus().TxSeen()[hash] == nil {
		node.Consensus().TxSeen()[hash] = make(map[string]bool)
	}
	// hardcoded max txs monitored
	if len(node.Consensus().TxSeen()) >= 32768 {
		// this adds indeterminism as range over maps is defined but not guaranteed indeterministic
		for k := range node.Consensus().TxSeen() {
			if k != hash {
				delete(node.Consensus().TxSeen(), k)
				return
			}
		}
	}
	node.Consensus().TxSeen()[hash][peerId] = true
}
