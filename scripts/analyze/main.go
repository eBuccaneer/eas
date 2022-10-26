package main

import (
	"encoding/csv"
	"encoding/json"
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"ethattacksim/util/helper"
	"ethattacksim/util/logger"
	"ethattacksim/util/stats"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	useAfterEip := false
	if len(os.Args) < 2 {
		log.Printf("Using pre EIP1559 stats. To specify post EIP1559 please use format './analyze[.exe] EIP1559 INPUT_DIR OUT_DIR' on command line.")
	} else {
		if os.Args[1] == "EIP1559" {
			useAfterEip = true
		}
		log.Printf("Using post EIP1559 stats: %v", useAfterEip)
	}

	dirName := "../../out"
	if len(os.Args) < 3 {
		log.Printf("Using standard input dir '%v'. To specify a different input dir please use format './analyze[.exe] [EIP1559 | NO_EIP1559] INPUT_DIR OUT_DIR' on command line.", dirName)
	} else {
		dirName = os.Args[2]
		log.Printf("Using file/dir '%v'.", dirName)
	}

	outDir := "./out"
	if len(os.Args) < 4 {
		log.Printf("Using standard output dir '%v'. To specify a different output dir please use format './analyze[.exe] [EIP1559 | NO_EIP1559] INPUT_DIR OUT_DIR' on command line.", outDir)
	} else {
		outDir = os.Args[3]
		log.Printf("Using outDir '%v'.", outDir)
	}

	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		panic("No files to process in directory " + dirName)
	}

	filesMode := false // if an input dir is given that directly contains auditLog.csv etc., than the for loop should only be executed once
	for _, f := range files {
		if !filesMode {
			var auditLogFileName string
			var overviewFileName string
			var outPath string
			if f.IsDir() {
				auditLogFileName = fmt.Sprintf("%v/%v/auditLog.csv", dirName, f.Name())
				overviewFileName = fmt.Sprintf("%v/%v/overview.json", dirName, f.Name())
				outPath = fmt.Sprintf("%v/%v", outDir, f.Name())
			} else if strings.HasSuffix(f.Name(), "auditLog.csv") || strings.HasSuffix(f.Name(), "overview.json") {
				auditLogFileName = fmt.Sprintf("%v/auditLog.csv", dirName)
				overviewFileName = fmt.Sprintf("%v/overview.json", dirName)
				filesMode = true
				outPath = outDir
			} else {
				continue
			}

			loggerFile := getOutFile("log_", auditLogFileName, outPath)
			log.SetOutput(logger.NewLogger(loggerFile, true))

			// result filled with dummy data at creation time, is filled in process functions
			result := &Result{0, 0, 0, 0, 0, 0, 0 /*, make(map[string]int)*/, 0, 0, 0 /*, make(map[string]int)*/, 0, 0, 0, 0, 0, 0 /*, make([]string, 0)*/, *NewHistogram(0, 10000000000, 250000000), make([][]HashrateAndRewards, 0, 1), make([]float64, 0), make([]float64, 0), make([]float64, 0), make([][]string, 0), make([]NodeStats, 0, 1), make([]float64, 0), make([]float64, 0), make([]float64, 0)}

			currentLongestChainBlockIds := make([]string, 0)

			isOverviewFile := checkIsFile(overviewFileName)
			if isOverviewFile {
				log.Printf("analyzing file %v...\n", overviewFileName)
				processOverview(overviewFileName, outPath, result, useAfterEip, &currentLongestChainBlockIds)
			} else {
				log.Panicf("Given file %v is not a file.", overviewFileName)
			}

			isAuditLogFile := checkIsFile(auditLogFileName)
			if isAuditLogFile {
				log.Printf("analyzing file %v...\n", auditLogFileName)
				processAuditLog(auditLogFileName, outPath, result, currentLongestChainBlockIds)
			} else {
				log.Panicf("Given file %v is not a file.", auditLogFileName)
			}

			PrintResult(result, outPath)
			_ = loggerFile.Close()
		}
	}
}

// the result that is written to json file
type Result struct {
	TimeSimulatedNanos                int     `json:"timeSimulated"`
	BlocksCount                       int     `json:"blocksCount"`
	MeanBlockPropagation              float64 `json:"meanBlockPropagationTime"` // mean time from mining till last node added block
	MedianBlockPropagation            float64 `json:"medianBlockPropagationTime"`
	StandardDeviationBlockPropagation float64 `json:"standardDeviationBlockPropagationTime"`
	MinBlockPropagation               int     `json:"minBlockPropagationTime"`
	MaxBlockPropagation               int     `json:"maxBlockPropagationTime"`
	//BlockPropagationPerBlock            map[string]int       `json:"blockPropagationPerBlock"` // time from mining till last node added specific block
	MeanBlockReception float64 `json:"meanBlockReceptionTime"` // mean time from mining till last node received block
	MinBlockReception  int     `json:"minBlockReceptionTime"`
	MaxBlockReception  int     `json:"maxBlockReceptionTime"`
	//BlockReceptionPerBlock              map[string]int       `json:"blockReceptionPerBlock"`        // time from mining till last node received specific block
	MeanBlockFirstPropagation float64 `json:"meanBlockFirstPropagationTime"` // mean time from mining till first node added block
	MeanBlockFirstReception   float64 `json:"meanBlockFirstReceptionTime"`   // mean time from mining till first node received block
	MinBlockFirstPropagation  int     `json:"minBlockFirstPropagationTime"`
	MaxBlockFirstPropagation  int     `json:"maxBlockFirstPropagationTime"`
	MinBlockFirstReception    int     `json:"minBlockFirstReceptionTime"`
	MaxBlockFirstReception    int     `json:"maxBlockFirstReceptionTime"`
	//NodeIds                             []string             `json:"nodeIds"`
	BlockPropagationHistogram                      Histogram              `json:"blockPropagationHistogram"`
	HashrateAndRewards                             [][]HashrateAndRewards `json:"hashrateRewards"`
	MedianDeviationHashrateAndRewards              []float64              `json:"medianDeviationHashrateRewardsPercentage"`
	MeanDeviationHashrateAndRewards                []float64              `json:"meanDeviationHashrateRewardsPercentage"`
	StandardDeviationHashrateAndRewards            []float64              `json:"standardDeviationHashrateRewardsPercentage"`
	HashrateAndRewardsSeenBy                       [][]string             `json:"hashrateRewardsSeenBy"`
	Stats                                          []NodeStats            `json:"stats"`
	MedianDeviationHashrateAndRewardsBlocksMined   []float64              `json:"medianDeviationRewardsBlocksMinedPercentage"`
	MeanDeviationHashrateAndRewardsBlocksMined     []float64              `json:"meanDeviationRewardsBlocksMinedPercentage"`
	StandardDeviationHashrateAndRewardsBlocksMined []float64              `json:"standardDeviationRewardsBlocksMinedPercentage"`
}

type TempBlockPropagation struct {
	Created        int
	BlockTimeStamp int
	Miner          string
	Received       map[string]int
	Finished       map[string]int
}

type HashrateAndRewards struct {
	SeenBy                      []string
	NodeId                      string
	HashRate                    float64
	HashRatePercentage          float64
	Rewards                     float64
	RewardsPercentage           float64
	DeviationHashrateRewards    float64
	MinedBlocks                 float64
	MinedBlocksPercentage       float64
	DeviationBlocksMinedRewards float64
}

type NodeStats struct {
	SeenBy         []string
	Blocks         int
	Uncles         int
	UnclesPerDay   float64
	Stales         int
	MeanBlockTime  float64
	Txs            int
	Throughput     float64
	BlockSizeMean  float64
	GasPriceMean   float64
	TxsPerDay      float64
	OverallRewards float64
}

type Histogram struct {
	Buckets        []int   `json:"buckets"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	RangePerBucket float64 `json:"rangePerBucket"`
}

func NewHistogram(min float64, max float64, rangePerBucket float64) *Histogram {
	size := int((max-min)/rangePerBucket + 2)
	hist := &Histogram{make([]int, size), min, max, rangePerBucket}
	for i, _ := range hist.Buckets {
		hist.Buckets[i] = 0
	}
	return hist
}

func (histogram *Histogram) AddEntry(value float64) {
	if value < histogram.Min {
		histogram.Buckets[0]++
	} else if value >= histogram.Max {
		histogram.Buckets[len(histogram.Buckets)-1]++
	} else {
		for i := 1; i < len(histogram.Buckets)-1; i++ {
			if float64(i-1)*histogram.RangePerBucket <= value && float64(i)*histogram.RangePerBucket > value {
				histogram.Buckets[i]++
				break
			}
		}
	}
}

func processOverview(fileName string, outPath string, result *Result, useAfterEip1559 bool, currentLongestChainBlockIds *[]string) {
	f := loadFile(fileName)
	byteValue, _ := ioutil.ReadAll(f)
	var overview stats.StatsOverview
	json.Unmarshal(byteValue, &overview)
	nodeCount := len(overview.StatsPerNodePerType)
	nodeStats := make(map[float64]NodeStats)
	hashratePerNode := make(map[string]HashrateAndRewards)
	for nId, statMap := range overview.StatsPerNodePerType {
		var newNodeStat NodeStats
		if useAfterEip1559 {
			newNodeStat = NodeStats{make([]string, 0, 1), int(statMap["current"]), int(statMap["uncles"]), statMap["unclesPerDay"], int(statMap["ledger"]) - int(statMap["current"]) - int(statMap["uncles"]), statMap["meanBlockTime"], int(statMap["txs"]), statMap["throughput"], statMap["meanBlockSize"], statMap["meanGasPrice"], 0, statMap["overallRewardsAfterEip1559"]}
		} else {
			newNodeStat = NodeStats{make([]string, 0, 1), int(statMap["current"]), int(statMap["uncles"]), statMap["unclesPerDay"], int(statMap["ledger"]) - int(statMap["current"]) - int(statMap["uncles"]), statMap["meanBlockTime"], int(statMap["txs"]), statMap["throughput"], statMap["meanBlockSize"], statMap["meanGasPrice"], 0, statMap["overallRewards"]}
		}
		newNodeStat.SeenBy = append(newNodeStat.SeenBy, nId)
		// something like a hash is used to differentiate different views in data set
		if _, exists := nodeStats[newNodeStat.Throughput*newNodeStat.MeanBlockTime]; exists {
			temp := nodeStats[newNodeStat.Throughput*newNodeStat.MeanBlockTime]
			temp.SeenBy = append(temp.SeenBy, nId)
			nodeStats[newNodeStat.Throughput*newNodeStat.MeanBlockTime] = temp
		} else {
			nodeStats[newNodeStat.Throughput*newNodeStat.MeanBlockTime] = newNodeStat
		}
		hashratePerNode[nId] = HashrateAndRewards{NodeId: nId, HashRate: statMap["hashRate"], HashRatePercentage: statMap["hashRatePercentage"]}
	}

	var numToBlockHash map[int]string
	for _, numToBlock := range overview.CurrentLedgerBlockIds {
		numToBlockHash = numToBlock
		break
	}

	for _, blockHash := range numToBlockHash {
		*currentLongestChainBlockIds = append(*currentLongestChainBlockIds, blockHash)
	}

	// compute hashrate and rewards comparison for different nodes views
	hashrateRewardsMap := make(map[float64][]HashrateAndRewards)
	deviationOverallMap := make(map[float64]float64)
	absDeviationsMap := make(map[float64][]float64)
	deviationBlocksMinedOverallMap := make(map[float64]float64)
	absDeviationsBlocksMinedMap := make(map[float64][]float64)
	seenByMap := make(map[float64][]string)
	var rewardsPerNodePerNode map[string]map[string]float64
	if useAfterEip1559 {
		rewardsPerNodePerNode = overview.RewardsPerNodePerNodeAfterEip1559
	} else {
		rewardsPerNodePerNode = overview.RewardsPerNodePerNode
	}
	for seenById, rewardsStat := range rewardsPerNodePerNode {
		haAndRe := make([]HashrateAndRewards, 0, nodeCount)
		deviationO := 0.0
		deviationOBlocksMined := 0.0
		absDev := make([]float64, 0, nodeCount)
		absDevBlocksMined := make([]float64, 0, nodeCount)
		seen := make([]string, 0, 1)
		seen = append(seen, seenById)
		allRewards := rewardsStat["ALL"]
		allMinedBlocks := overview.BlockCountPerNode[seenById]
		for nodeId, nodeReward := range rewardsStat {
			if nodeId == "GENESIS" || nodeId == "ALL" {
				continue
			}
			rewardsPercentage := nodeReward / allRewards * 100
			deviation := rewardsPercentage - hashratePerNode[nodeId].HashRatePercentage
			deviationO += math.Abs(deviation)
			absDev = append(absDev, math.Abs(deviation))
			minedBlocks := overview.MinedBlockCountPerNodePerNode[seenById][nodeId]
			deviationBlocksMined := rewardsPercentage - float64(minedBlocks)/float64(allMinedBlocks)*100
			deviationOBlocksMined += math.Abs(deviationBlocksMined)
			absDevBlocksMined = append(absDevBlocksMined, math.Abs(deviationBlocksMined))
			haR := HashrateAndRewards{NodeId: nodeId, HashRate: hashratePerNode[nodeId].HashRate, HashRatePercentage: hashratePerNode[nodeId].HashRatePercentage, Rewards: nodeReward, RewardsPercentage: rewardsPercentage, DeviationHashrateRewards: deviation, MinedBlocks: float64(minedBlocks), MinedBlocksPercentage: float64(minedBlocks) / float64(allMinedBlocks) * 100, DeviationBlocksMinedRewards: deviationBlocksMined}
			haAndRe = append(haAndRe, haR)
		}

		for nodeId := range hashratePerNode {
			if nodeId == "GENESIS" || nodeId == "ALL" {
				continue
			}

			found := false
			for _, hR := range haAndRe {
				if hR.NodeId == nodeId {
					found = true
					break
				}
			}

			if found {
				// only add nodes that are not included already and therefore have won 0 blocks and rewards
				continue
			}

			nodeReward := 0.0
			rewardsPercentage := nodeReward / allRewards * 100
			deviation := rewardsPercentage - hashratePerNode[nodeId].HashRatePercentage
			deviationO += math.Abs(deviation)
			absDev = append(absDev, math.Abs(deviation))
			minedBlocks := overview.MinedBlockCountPerNodePerNode[seenById][nodeId]
			deviationBlocksMined := rewardsPercentage - float64(minedBlocks)/float64(allMinedBlocks)*100
			deviationOBlocksMined += math.Abs(deviationBlocksMined)
			absDevBlocksMined = append(absDevBlocksMined, math.Abs(deviationBlocksMined))
			haR := HashrateAndRewards{NodeId: nodeId, HashRate: hashratePerNode[nodeId].HashRate, HashRatePercentage: hashratePerNode[nodeId].HashRatePercentage, Rewards: nodeReward, RewardsPercentage: rewardsPercentage, DeviationHashrateRewards: deviation, MinedBlocks: float64(minedBlocks), MinedBlocksPercentage: float64(minedBlocks) / float64(allMinedBlocks) * 100, DeviationBlocksMinedRewards: deviationBlocksMined}
			haAndRe = append(haAndRe, haR)
		}

		sort.Slice(haAndRe, func(i, j int) bool {
			return haAndRe[i].HashRate > haAndRe[j].HashRate
		})
		hash := 0.0
		for i := 0; i < int(math.Min(float64(len(haAndRe)), 20)); i++ {
			// checks the first 20 nodes for overall equality in rewards and rewards percentage
			hash += haAndRe[i].Rewards * haAndRe[i].RewardsPercentage
		}
		if _, exists := seenByMap[hash]; exists {
			temp := seenByMap[hash]
			temp = append(temp, seenById)
			seenByMap[hash] = temp
		} else {
			hashrateRewardsMap[hash] = haAndRe
			deviationOverallMap[hash] = deviationO
			absDeviationsMap[hash] = absDev
			deviationBlocksMinedOverallMap[hash] = deviationOBlocksMined
			absDeviationsBlocksMinedMap[hash] = absDevBlocksMined
			seenByMap[hash] = seen
		}
	}

	for hash, rewardsSight := range hashrateRewardsMap {
		absDeviations := absDeviationsMap[hash]
		absDeviationsBlocksMined := absDeviationsBlocksMinedMap[hash]
		sort.Float64s(absDeviations)
		sort.Float64s(absDeviationsBlocksMined)
		if len(absDeviations)%2 == 0 {
			result.MedianDeviationHashrateAndRewards = append(result.MedianDeviationHashrateAndRewards, (absDeviations[len(absDeviations)/2-1]+absDeviations[len(absDeviations)/2])/2)
			result.MedianDeviationHashrateAndRewardsBlocksMined = append(result.MedianDeviationHashrateAndRewardsBlocksMined, (absDeviationsBlocksMined[len(absDeviationsBlocksMined)/2-1]+absDeviationsBlocksMined[len(absDeviationsBlocksMined)/2])/2)
		} else {
			result.MedianDeviationHashrateAndRewards = append(result.MedianDeviationHashrateAndRewards, absDeviations[int(math.Floor(float64(len(absDeviations))/2))])
			result.MedianDeviationHashrateAndRewardsBlocksMined = append(result.MedianDeviationHashrateAndRewardsBlocksMined, absDeviationsBlocksMined[int(math.Floor(float64(len(absDeviationsBlocksMined))/2))])
		}

		result.HashrateAndRewards = append(result.HashrateAndRewards, rewardsSight)
		deviationOverall := deviationOverallMap[hash]
		deviationMean := deviationOverall / float64(nodeCount)
		deviationOverallBlocksMined := deviationBlocksMinedOverallMap[hash]
		deviationMeanBlocksMined := deviationOverallBlocksMined / float64(nodeCount)
		result.MeanDeviationHashrateAndRewards = append(result.MeanDeviationHashrateAndRewards, deviationMean)
		result.MeanDeviationHashrateAndRewardsBlocksMined = append(result.MeanDeviationHashrateAndRewardsBlocksMined, deviationMeanBlocksMined)
		variance := 0.0
		varianceBlocksMined := 0.0
		for _, elem := range rewardsSight {
			variance += math.Pow(math.Abs(elem.DeviationHashrateRewards)-deviationMean, 2)
			varianceBlocksMined += math.Pow(math.Abs(elem.DeviationBlocksMinedRewards)-deviationMean, 2)
		}
		variance /= float64(nodeCount)
		varianceBlocksMined /= float64(nodeCount)
		result.StandardDeviationHashrateAndRewards = append(result.StandardDeviationHashrateAndRewards, math.Sqrt(variance))
		result.StandardDeviationHashrateAndRewardsBlocksMined = append(result.StandardDeviationHashrateAndRewardsBlocksMined, math.Sqrt(varianceBlocksMined))
		result.HashrateAndRewardsSeenBy = append(result.HashrateAndRewardsSeenBy, seenByMap[hash])
	}

	simTime := overview.SimulatedTime
	result.TimeSimulatedNanos = int(simTime)
	simDays := float64(simTime) / 86400000000000
	for _, nodeStat := range nodeStats {
		nodeStat.TxsPerDay = float64(nodeStat.Txs) / simDays
		result.Stats = append(result.Stats, nodeStat)
	}
}

func processAuditLog(fileName string, outPath string, result *Result, currentLongestChainBlockIds []string) {
	// read file and process all rows
	f := loadFile(fileName)
	reader := csv.NewReader(f)
	reader.Comma = ';'
	reader.Read() // first line is header and not needed
	tempResults := make(map[string]*TempBlockPropagation)
	nodeMap := make(map[string]bool)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		processAuditLogRow(record, tempResults, nodeMap)
	}

	// create result
	// result := &Result{0, 0, 0, 0, make(map[string]int), 0, 0, 0, make(map[string]int), 0, 0, 0, 0, 0, 0, make([]string, 0), *NewHistogram(0, 10000000000, 250000000)}
	finishedBlockCount := 0
	propagationTimes := make([]int, 0, 0)
	blockCreationTimes := make([]int, 0, 0)
	blockCreationTimesWithUncles := make([]int, 0, 0)
	meanPropagation := 0.0
	minPropagationTime := math.MaxInt64
	maxPropagationTime := 0
	meanReception := 0.0
	minReceptionTime := math.MaxInt64
	maxReceptionTime := 0
	meanFirstPropagation := 0.0
	meanFirstReception := 0.0
	minFirstPropagationTime := math.MaxInt64
	maxFirstPropagationTime := 0
	minFirstReceptionTime := math.MaxInt64
	maxFirstReceptionTime := 0
	blockPropagationHistogram := NewHistogram(0, 10000000000, 250000000)

	// gather node ids that occur in the log
	nodeIds := make([]string, 0)
	for nodeId, _ := range nodeMap {
		nodeIds = append(nodeIds, nodeId)
	}
	//result.NodeIds = nodeIds

	receptionMaxLogFile := getOutFile("reception_max_", fileName, outPath)
	defer receptionMaxLogFile.Close()
	blockPropagationMaxFile := getOutFile("blockPropagation_max_", fileName, outPath)
	defer blockPropagationMaxFile.Close()
	receptionLogFile := getOutFile("reception_", fileName, outPath)
	defer receptionLogFile.Close()
	blockPropagationFile := getOutFile("blockPropagation_", fileName, outPath)
	defer blockPropagationFile.Close()
	blockTimeFile := getOutFile("blockTime_", fileName, outPath)
	defer blockTimeFile.Close()
	blockTimeWithUnclesFile := getOutFile("blockTimeWithUncles_", fileName, outPath)
	defer blockTimeWithUnclesFile.Close()

	// process temp results
	for id, tempResult := range tempResults {
		blockCreationTimesWithUncles = append(blockCreationTimesWithUncles, tempResult.BlockTimeStamp)
		if helper.ContainsString(currentLongestChainBlockIds, id) {
			blockCreationTimes = append(blockCreationTimes, tempResult.BlockTimeStamp)
		}
		finished := len(tempResult.Finished) == len(nodeMap)-1 // -1 because miner is excluded
		if !finished {
			// if not all nodes are finished receiving a block, check which are not and print to log
			log.Printf("Block '%v' not finished yet:\n", id)
			for _, nodeId := range nodeIds {
				if _, exists := tempResult.Finished[nodeId]; !exists {
					log.Printf("\tNode '%v' not finished with block '%v' \n", nodeId, id)
				}
			}
		} else {
			// process all blocks that all nodes are finished with
			finishedBlockCount++
			lastFinished := 0
			firstFinished := math.MaxInt64
			for nodeId, time := range tempResult.Finished {
				printLine(fmt.Sprintf("%v", time-tempResult.Created), blockPropagationFile)
				blockPropagationHistogram.AddEntry(float64(time - tempResult.Created))
				if time-tempResult.Created < 0 {
					log.Printf("!!! Block '%v' at node '%v' finished in - time '%v', created '%v', finished '%v' \n", id, nodeId, time-tempResult.Created, tempResult.Created, time)
				}
				if time > lastFinished {
					lastFinished = time
				}
				if time < firstFinished {
					firstFinished = time
				}
			}
			propTime := lastFinished - tempResult.Created
			firstPropTime := firstFinished - tempResult.Created
			meanFirstPropagation += float64(firstPropTime)
			printLine(fmt.Sprintf("%v", propTime), blockPropagationMaxFile)
			propagationTimes = append(propagationTimes, propTime)
			if propTime > maxPropagationTime {
				maxPropagationTime = propTime
			}
			if propTime < minPropagationTime {
				minPropagationTime = propTime
			}
			if firstPropTime > maxFirstPropagationTime {
				maxFirstPropagationTime = firstPropTime
			}
			if firstPropTime < minFirstPropagationTime {
				minFirstPropagationTime = firstPropTime
			}
			meanPropagation += float64(propTime)
			//result.BlockPropagationPerBlock[id] = propTime

			lastReception := 0
			firstReception := math.MaxInt64
			for _, time := range tempResult.Received {
				printLine(fmt.Sprintf("%v", time-tempResult.Created), receptionLogFile)
				if time > lastReception {
					lastReception = time
				}
				if time < firstReception {
					firstReception = time
				}
			}
			recTime := lastReception - tempResult.Created
			firstRecTime := firstReception - tempResult.Created
			meanFirstReception += float64(firstRecTime)
			printLine(fmt.Sprintf("%v", recTime), receptionMaxLogFile)
			if recTime > maxReceptionTime {
				maxReceptionTime = recTime
			}
			if recTime < minReceptionTime {
				minReceptionTime = recTime
			}
			if firstRecTime > maxFirstReceptionTime {
				maxFirstReceptionTime = firstRecTime
			}
			if firstRecTime < minFirstReceptionTime {
				minFirstReceptionTime = firstRecTime
			}
			meanReception += float64(recTime)
			//result.BlockReceptionPerBlock[id] = recTime
		}
	}

	// log block times
	sort.Ints(blockCreationTimes)
	sort.Ints(blockCreationTimesWithUncles)
	lastNanos := 0
	for _, nanos := range blockCreationTimes {
		if lastNanos != 0 {
			printLine(fmt.Sprintf("%d", (nanos/1000000000)-(lastNanos/1000000000)), blockTimeFile)
		}
		lastNanos = nanos
	}
	lastNanos = 0
	for _, nanos := range blockCreationTimesWithUncles {
		if lastNanos != 0 {
			printLine(fmt.Sprintf("%d", (nanos/1000000000)-(lastNanos/1000000000)), blockTimeWithUnclesFile)
		}
		lastNanos = nanos
	}

	// compute stats
	sort.Ints(propagationTimes)
	if len(propagationTimes)%2 == 0 {
		result.MedianBlockPropagation = (float64(propagationTimes[len(propagationTimes)/2-1]) + float64(propagationTimes[len(propagationTimes)/2])) / 2
	} else {
		result.MedianBlockPropagation = float64(propagationTimes[int(math.Floor(float64(len(propagationTimes))/2))])
	}
	result.MeanBlockPropagation = meanPropagation / float64(finishedBlockCount)

	variance := 0.0
	for _, elem := range propagationTimes {
		variance += math.Pow(float64(elem)-result.MeanBlockPropagation, 2)
	}
	variance /= float64(len(propagationTimes))
	result.StandardDeviationBlockPropagation = math.Sqrt(variance)

	result.BlocksCount = finishedBlockCount
	result.MinBlockPropagation = minPropagationTime
	result.MaxBlockPropagation = maxPropagationTime
	result.MeanBlockReception = meanReception / float64(finishedBlockCount)
	result.MinBlockReception = minReceptionTime
	result.MaxBlockReception = maxReceptionTime
	result.MeanBlockFirstPropagation = meanFirstPropagation / float64(finishedBlockCount)
	result.MeanBlockFirstReception = meanFirstReception / float64(finishedBlockCount)
	result.MinBlockFirstPropagation = minFirstPropagationTime
	result.MaxBlockFirstPropagation = maxFirstPropagationTime
	result.MinBlockFirstReception = minFirstReceptionTime
	result.MaxBlockFirstReception = maxFirstReceptionTime
	result.BlockPropagationHistogram = *blockPropagationHistogram
}

func processAuditLogRow(record []string, tempResults map[string]*TempBlockPropagation, nodeMap map[string]bool) {
	// read the row from a csv file
	time, _ := strconv.Atoi(strings.TrimSpace(record[0]))
	nodeId := strings.TrimSpace(record[1])
	eventType := strings.TrimSpace(record[2])
	fromTo := strings.TrimSpace(record[3])
	id := strings.TrimSpace(record[4])
	//text := strings.TrimSpace(record[5])

	from := ""
	to := ""

	// track all nodes that occur in the log
	nodeMap[nodeId] = true

	if len(fromTo) > 0 {
		fromToSplit := strings.Split(fromTo, "->")
		from = strings.TrimSpace(fromToSplit[0])
		to = strings.TrimSpace(fromToSplit[1])
		// track all nodes that occur in the log
		nodeMap[from] = true
		nodeMap[to] = true
	}

	switch {
	// if a new block event occurs, create an instance of TempBlockPropagation for it with the time the block was created
	case eventType == interfaces.NEW_BLOCK_EVENT.String():
		tempResults[id] = &TempBlockPropagation{time, 0, nodeId, make(map[string]int), make(map[string]int)}
	// if a new block event occurs, create an instance of TempBlockPropagation for it with the time the block was created
	case eventType == interfaces.NEW_BLOCK_TIMESTAMP.String():
		tempResults[id].BlockTimeStamp = time
	// if a block was received or block bodies were received, log time to TempBlockPropagation (only if receiver of event was nodeId)
	case eventType == interfaces.RECEIVED_BLOCK_EVENT.String():
		if nodeId != to || nodeId == tempResults[id].Miner {
			break
		}
		fallthrough
	case eventType == interfaces.RECEIVED_BLOCK_BODIES_EVENT.String():
		blockHashesSplit := strings.Split(id, ",")
		for _, blockHash := range blockHashesSplit {
			tempResult := tempResults[blockHash]
			_, isFinished := tempResult.Finished[nodeId]
			if tempResult != nil && !isFinished { // only count last received time if node has not written block yet as some events may occur afterwards
				if nodeId != to || nodeId == tempResult.Miner {
					break
				}
				if _, firstReceivedSet := tempResult.Received[nodeId]; !firstReceivedSet {
					tempResult.Received[nodeId] = time
				}
			}
		}
	// if the blocks were processed on the node side, log to TempBlockPropagation
	case eventType == "BLOCK_APPENDED":
		fallthrough
	case eventType == "BLOCK_WRITTEN":
		fallthrough
	case eventType == "CHAIN_REORG_BLOCK_WRITTEN":
		fallthrough
	case eventType == "SIDECHAIN_BLOCK_APPENDED":
		fallthrough
	case eventType == "SIDECHAIN_BLOCK_WRITTEN":
		fallthrough
	case eventType == "SIDECHAIN_REORG_BLOCK_WRITTEN":
		fallthrough
	case eventType == "FUTURE_BLOCK":
		fallthrough
	case eventType == "BLOCK_OLD":
		fallthrough
	case eventType == "UNKNOWN_PARENT":
		fallthrough
	case eventType == "INVALID_HEADER":
		fallthrough
	case eventType == "IMPORT_FAILED":
		fallthrough
	case eventType == "UNKNOWN_BLOCK_ERROR":
		fallthrough
	case eventType == "UNKNOWN_BLOCK_STATUS":
		if _, finishedSet := tempResults[id].Finished[nodeId]; !finishedSet && nodeId != tempResults[id].Miner {
			tempResults[id].Finished[nodeId] = time // mark the node as finished when the block was processed
		}
	default:
		//log.Printf("\tNode '%v' block '%v', event '%v' not processed \n", nodeId, id, eventType)
	}
}

func checkIsFile(name string) bool {
	f, err := os.Stat(name)
	if err != nil {
		log.Panic(err)
	}
	switch mode := f.Mode(); {
	case mode.IsDir():
		return false
	case mode.IsRegular():
		return true
	default:
		panic("Input file is neither file nor directory.")
	}
}

func loadFile(name string) *os.File {
	f, err := os.Open(name)
	if err != nil {
		log.Panic(err)
	}
	return f
}

func printLine(line string, f *os.File) {
	if f != nil {
		// write csv headers
		_, _ = f.Write([]byte(fmt.Sprintf("%v\n", line)))
	}
}

func getOutFile(prefix string, name string, outPath string) *os.File {
	_, fileName := filepath.Split(name)
	fileNameSplit := strings.Split(fileName, ".")
	outFile := filepath.Join(outPath, prefix+fileNameSplit[0]+".txt")
	if file.FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		file.EnsureOutPath(outPath)
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}
	return outputFile
}

func PrintResult(result *Result, outPath string) {
	statsOverview, _ := json.Marshal(result)
	outFile := filepath.Join(outPath, "result.json")
	//outFile := outPath + "/result.json"
	if file.FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		file.EnsureOutPath(outPath)
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}
	outputFile.Write(statsOverview)
	_ = outputFile.Close()
}
