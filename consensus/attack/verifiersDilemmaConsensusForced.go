package attack

import (
	"crypto/sha256"
	"encoding/base64"
	"ethattacksim/event"
	"ethattacksim/event/events"
	"ethattacksim/interfaces"
	ledg "ethattacksim/ledger"
	"ethattacksim/util/helper"
	"ethattacksim/util/random"
	"fmt"
	"sort"
	"strings"
)

type VerifiersDilemmaConsensusForced struct {
	interfaces.IConsensus
}

func NewVerifiersDilemmaConsensusForced(consensus interfaces.IConsensus) interfaces.IConsensus {
	return &VerifiersDilemmaConsensusForced{IConsensus: consensus}
}

func (c *VerifiersDilemmaConsensusForced) VerifyTx(tx interfaces.ITransaction, node interfaces.INode) (ok bool) {
	return true
}

func (c *VerifiersDilemmaConsensusForced) VerifyState(block interfaces.IBlock, node interfaces.INode, checkPastTx bool) (ok bool) {
	return true
}

func (c *VerifiersDilemmaConsensusForced) BroadcastReceivedBlockTargets(node interfaces.INode, block interfaces.IBlock, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	return
}

func (c *VerifiersDilemmaConsensusForced) BroadcastNewBlockTargets(node interfaces.INode, block interfaces.IBlock, propagate bool, excludeIds ...string) (targets []interfaces.INode) {
	selectedPeers := make([]interfaces.INode, 0, len(node.Peers()))
	for _, peer := range node.Peers() {
		if _, ok := node.Consensus().BlockSeen()[block.Hash()][peer.Id()]; !ok {
			if !helper.ContainsString(excludeIds, peer.Id()) {
				selectedPeers = append(selectedPeers, peer)
			}
		}
	}
	if propagate {
		// send whole block to all peers immediately
		targets = selectedPeers
	} else {
		targets = make([]interfaces.INode, 0)
	}
	for _, target := range targets {
		node.Consensus().MarkBlockSeen(node, block.Hash(), target.Id())
	}
	return
}

func (c *VerifiersDilemmaConsensusForced) MineBlock(ledger interfaces.ILedger, node interfaces.INode, world interfaces.IWorld) {
	blockTimeStamp, miningTimeDelay := random.TimeBetweenBlocks(world.SimConfig().OverallHashPower(), node.HashPower(), ledger.Head(node).Header().Time(), node.Time())
	miningTime := node.Time() + miningTimeDelay
	newGasLimit := node.Consensus().GetGasLimit(ledger.Head(node).Header(), world)
	txs := make([]interfaces.ITransaction, 0, 50)
	gasAlreadyUsed := 0

	// create bad transaction
	if world.SimConfig().Attacker().Numbers()["percentOfGasToForceVerifiersDilemma"] > 0 {
		badTx := createBadTransaction(int(float64(newGasLimit)*world.SimConfig().Attacker().Numbers()["percentOfGasToForceVerifiersDilemma"]), world.SimConfig().Attacker().Numbers()["specialTxStateComputation"], node)
		txs = append(txs, badTx)
		gasAlreadyUsed += badTx.GasUsed()
	}

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

func createBadTransaction(gasToUse int, specialTxStateComputation float64, senderNode interfaces.INode) interfaces.ITransaction {
	senderId, senderNonce := senderNode.Id(), senderNode.Nonce()
	senderNode.IncNonce()
	return ledg.NewTx(fmt.Sprintf("R%v_%v", senderId, senderNonce), senderNonce, senderId, gasToUse, int(random.GasPrice()), true, specialTxStateComputation)
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

func (c *VerifiersDilemmaConsensusForced) GetGasLimit(currentHead interfaces.IBlockHeader, world interfaces.IWorld) (gasLimit int) {
	maxAdaption := currentHead.GasLimit() / 1024 // maximum allowed change of gas limit
	maxAdaption = int(float64(maxAdaption) * world.SimConfig().Attacker().Numbers()["percentOfMaxGasLimitIncrease"])
	gasLimit = currentHead.GasLimit() + maxAdaption
	return
}
