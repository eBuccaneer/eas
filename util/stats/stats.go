package stats

import (
	"encoding/json"
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"log"
	"math"
	"os"
	"sort"
)

func PrintWorld(world interfaces.IWorld, file *os.File) {
	stats, _ := json.Marshal(world)
	file.Write(stats)
}

func PrintStatsOverview(world interfaces.IWorld, file *os.File, config *file.Config) {
	statsOverview, _ := json.Marshal(NewStatsOverview(world, config))
	file.Write(statsOverview)
}

type StatsOverview struct {
	SimulatedTime                     int64
	BlockCountPerNode                 map[string]int
	MinedBlockCountPerNodePerNode     map[string]map[string]int
	RewardsPerNodePerNode             map[string]map[string]float64 // eth
	RewardsPerNodePerNodeAfterEip1559 map[string]map[string]float64 // eth
	StatsPerNodePerType               map[string]map[string]float64
	PeersPerNode                      map[string]string
	CurrentLedgerBlockIds             map[string]map[int]string
}

func NewStatsOverview(world interfaces.IWorld, config *file.Config) *StatsOverview {
	var minedBlockCount map[string]map[string]int = make(map[string]map[string]int)
	var rewardsPerNodePerNode map[string]map[string]float64 = make(map[string]map[string]float64)
	var rewardsPerNodePerNodeAfterEip1559 map[string]map[string]float64 = make(map[string]map[string]float64)
	var blockCount map[string]int = make(map[string]int)
	var statsPerNodePerType map[string]map[string]float64 = make(map[string]map[string]float64)
	var peersPerNode map[string]string = make(map[string]string)
	var currentLedgerBlockIdsPerNode map[string]map[int]string = make(map[string]map[int]string)

	// use sorted node key array because of determinism
	var nodeIds []string = make([]string, 0, len(world.Nodes()))
	for nId, _ := range world.Nodes() {
		nodeIds = append(nodeIds, nId)
	}
	sort.Strings(nodeIds)

	overallRewardsArr := make([]float64, 0)
	// compute stats
	for _, nId := range nodeIds {
		n := world.Nodes()[nId]

		blockCount[n.Id()] = n.Ledger().Length(n)
		if _, ok := statsPerNodePerType[n.Id()]; !ok {
			statsPerNodePerType[n.Id()] = make(map[string]float64)
		}
		statsPerNodePerType[n.Id()]["current"] = float64(n.Ledger().Length(n))
		statsPerNodePerType[n.Id()]["ledger"] = float64(len(n.Ledger().Get()))
		statsPerNodePerType[n.Id()]["uncles"] = float64(len(n.Ledger().Uncles()))
		peerString := ""
		for i, peer := range n.Peers() {
			peerString += peer.Id()
			if i < len(n.Peers())-1 {
				peerString += ","
			}
		}
		peersPerNode[nId] = peerString
		expectedNum := 0

		if minedBlockCount[n.Id()] == nil {
			minedBlockCount[n.Id()] = make(map[string]int)
		}
		if rewardsPerNodePerNode[n.Id()] == nil {
			rewardsPerNodePerNode[n.Id()] = make(map[string]float64)
		}
		if rewardsPerNodePerNodeAfterEip1559[n.Id()] == nil {
			rewardsPerNodePerNodeAfterEip1559[n.Id()] = make(map[string]float64)
		}
		if n.Id() == "node_pool1" {
			if currentLedgerBlockIdsPerNode[n.Id()] == nil {
				currentLedgerBlockIdsPerNode[n.Id()] = make(map[int]string)
			}
		}
		txCount := 0
		blockSizeCumulated := 0
		gasPriceCumulated := 0
		for i, block := range n.Ledger().CurrentLedgerByHeight() {
			if expectedNum != block.Header().Number() {
				log.Printf("error with block numbers")
			} else {
				expectedNum++
			}

			if n.Id() == "node_pool1" {
				currentLedgerBlockIdsPerNode[n.Id()][i] = block.Hash()
			}
			minedBlockCount[n.Id()][block.Header().MinerId()] += 1

			txCount += len(block.Body().Transactions())
			blockSizeCumulated += block.Header().Size()

			// block reward
			rewardsPerNodePerNode[n.Id()][block.Header().MinerId()] += config.BlockReward()
			rewardsPerNodePerNodeAfterEip1559[n.Id()][block.Header().MinerId()] += config.BlockReward()

			if len(block.Body().Uncles()) > 0 {
				// nephew reward
				rewardsPerNodePerNode[n.Id()][block.Header().MinerId()] += config.BlockNephewReward()
				rewardsPerNodePerNodeAfterEip1559[n.Id()][block.Header().MinerId()] += config.BlockNephewReward()

				// uncle reward
				for _, uncleBlockHeader := range block.Body().Uncles() {
					if uncleBlockHeader.Number()+7-block.Header().Number() < 0 {
						log.Printf("error with uncles")
					}
					uncleReward := float64(uncleBlockHeader.Number()+8-block.Header().Number()) * config.BlockReward() / 8
					rewardsPerNodePerNode[n.Id()][uncleBlockHeader.MinerId()] += uncleReward
					rewardsPerNodePerNodeAfterEip1559[n.Id()][uncleBlockHeader.MinerId()] += uncleReward
				}
			}

			// tx costs reward
			gasReward := 0.0
			gasRewardEip1559 := 0.0
			for _, tx := range block.Body().Transactions() {
				gasPriceCumulated += tx.GasPrice()
				// compute rewards if sender is not miner
				if tx.SenderId() != block.Header().MinerId() {
					gasReward += float64(tx.GasUsed()*tx.GasPrice()) / 1000000000 // gwei to eth
				}

				if tx.SenderId() == block.Header().MinerId() {
					// here the gas is deducted for post EIP1559 transaction
					// but only for the ones in the own block, these are the ones interesting us for i.e. verifiers dilemma as they are not broadcasted
					gasRewardEip1559 -= float64(tx.GasUsed()*tx.GasPrice()) / 1000000000 // gwei to eth
				}
			}
			//log.Printf("%v - gas %v - gasEip1559 %v", block.Header().MinerId(), gasReward, gasRewardEip1559)
			rewardsPerNodePerNode[n.Id()][block.Header().MinerId()] += gasReward
			rewardsPerNodePerNodeAfterEip1559[n.Id()][block.Header().MinerId()] += gasRewardEip1559
		}

		overallRewards := 0.0
		overallRewardsAfterEip1559 := 0.0
		for _, value := range rewardsPerNodePerNode[n.Id()] {
			overallRewards += value
		}
		for _, value := range rewardsPerNodePerNodeAfterEip1559[n.Id()] {
			overallRewardsAfterEip1559 += value
		}
		rewardsPerNodePerNode[n.Id()]["ALL"] = getNearestEqual(&overallRewardsArr, overallRewards)
		rewardsPerNodePerNodeAfterEip1559[n.Id()]["ALL"] = getNearestEqual(&overallRewardsArr, overallRewardsAfterEip1559)

		statsPerNodePerType[n.Id()]["txs"] = float64(txCount)
		timeSimulated := world.Time()
		secondsSimulated := float64(timeSimulated) / 1000000000
		daysSimulated := float64(timeSimulated) / 1000000000 / 60 / 60 / 24
		statsPerNodePerType[n.Id()]["throughput"] = float64(txCount) / secondsSimulated // tx/s
		statsPerNodePerType[n.Id()]["unclesPerDay"] = statsPerNodePerType[n.Id()]["uncles"] / daysSimulated
		statsPerNodePerType[n.Id()]["meanBlockTime"] = secondsSimulated / statsPerNodePerType[n.Id()]["current"]   // seconds
		statsPerNodePerType[n.Id()]["meanBlockSize"] = float64(blockSizeCumulated) / float64(n.Ledger().Length(n)) // bytes
		statsPerNodePerType[n.Id()]["meanGasPrice"] = float64(gasPriceCumulated) / float64(txCount)                // gwei
		statsPerNodePerType[n.Id()]["hashRate"] = n.HashPower()                                                    // MH
		statsPerNodePerType[n.Id()]["hashRatePercentage"] = n.HashPower() / world.SimConfig().OverallHashPower() * 100
		statsPerNodePerType[n.Id()]["rewards"] = rewardsPerNodePerNode[n.Id()][n.Id()] // eth
		statsPerNodePerType[n.Id()]["rewardsPercentage"] = rewardsPerNodePerNode[n.Id()][n.Id()] / overallRewards * 100
		statsPerNodePerType[n.Id()]["rewardsAfterEip1559"] = rewardsPerNodePerNodeAfterEip1559[n.Id()][n.Id()] // eth
		statsPerNodePerType[n.Id()]["rewardsPercentageAfterEip1559"] = rewardsPerNodePerNodeAfterEip1559[n.Id()][n.Id()] / overallRewardsAfterEip1559 * 100
		statsPerNodePerType[n.Id()]["minedBlocks"] = float64(minedBlockCount[n.Id()][n.Id()])
		statsPerNodePerType[n.Id()]["overallRewards"] = overallRewards
		statsPerNodePerType[n.Id()]["overallRewardsAfterEip1559"] = overallRewardsAfterEip1559
	}

	return &StatsOverview{world.Time(), blockCount, minedBlockCount, rewardsPerNodePerNode, rewardsPerNodePerNodeAfterEip1559, statsPerNodePerType, peersPerNode, currentLedgerBlockIdsPerNode}
}

const float64EqualityThreshold = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

// this is a workaround only used with overall rewards comparison, because of errors in float64
func getNearestEqual(overallRewardsArr *[]float64, val float64) float64 {
	for _, v := range *overallRewardsArr {
		if almostEqual(val, v) {
			return v
		}
	}
	*overallRewardsArr = append(*overallRewardsArr, val)
	return val
}
