package main

import (
	"encoding/json"
	"ethattacksim/util/file"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dirName := "../analyze/out"
	if len(os.Args) < 2 {
		log.Printf("Using standard input dir '%v'. To specify a different input dir please use format './summarize[.exe] INPUT_DIR OUT_DIR' on command line.", dirName)
	} else {
		dirName = os.Args[1]
		log.Printf("Using file/dir '%v'.", dirName)
	}
	outDir := "./out"
	if len(os.Args) < 3 {
		log.Printf("Using standard output dir '%v'. To specify a different output dir please use format './summarize[.exe] INPUT_DIR OUT_DIR' on command line.", outDir)
	} else {
		outDir = os.Args[2]
		log.Printf("Using outDir '%v'.", outDir)
	}

	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		panic("No files to process in directory " + dirName)
	}

	summaryResultFile := getOutFile(outDir, "summary.csv")
	// write header
	_, _ = summaryResultFile.Write([]byte(fmt.Sprintf("%v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v\n", "seed", "time simulated nanos", "blocks", "block time mean", "stale blocks", "uncles", "uncles/day", "tx", "tx/day", "throughput (tx/s)", "block size mean", "gas price mean", "overall rewards", "deviationHashrateRewardsPercentage mean", "deviationHashrateRewardsPercentage median", "deviationHashrateRewardsPercentage sd", "deviationBlocksMinedRewardsPercentage mean", "deviationBlocksMinedRewardsPercentage median", "deviationBlocksMinedRewardsPercentage sd", "blockPropagation mean", "blockPropagation median", "blockPropagation sd", "current ledger length")))

	for _, f := range files {
		var resultFileName string
		if f.IsDir() {
			resultFileName = fmt.Sprintf("%v/%v/result.json", dirName, f.Name())
		} else {
			continue
		}

		isResultFile := checkIsFile(resultFileName)
		if isResultFile {
			log.Printf("analyzing file %v...\n", resultFileName)
			processSummary(resultFileName, summaryResultFile, outDir, f.Name())
		} else {
			log.Panicf("Given file %v is not a file.", resultFileName)
		}
	}
}

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

func processSummary(fileName string, outFile *os.File, outDir string, seed string) {
	f := loadFile(fileName)
	byteValue, _ := ioutil.ReadAll(f)
	var result Result
	json.Unmarshal(byteValue, &result)
	_, _ = outFile.Write([]byte(fmt.Sprintf("%v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v ; %v\n", seed, convert(result.TimeSimulatedNanos), convert(result.BlocksCount), convert(result.Stats[0].MeanBlockTime), convert(result.Stats[0].Stales), convert(result.Stats[0].Uncles), convert(result.Stats[0].UnclesPerDay), convert(result.Stats[0].Txs), convert(result.Stats[0].TxsPerDay), convert(result.Stats[0].Throughput), convert(result.Stats[0].BlockSizeMean), convert(result.Stats[0].GasPriceMean), convert(result.Stats[0].OverallRewards), convert(result.MeanDeviationHashrateAndRewards[0]), convert(result.MedianDeviationHashrateAndRewards[0]), convert(result.StandardDeviationHashrateAndRewards[0]), convert(result.MeanDeviationHashrateAndRewardsBlocksMined[0]), convert(result.MedianDeviationHashrateAndRewardsBlocksMined[0]), convert(result.StandardDeviationHashrateAndRewardsBlocksMined[0]), convert(result.MeanBlockPropagation), convert(result.MedianBlockPropagation), convert(result.StandardDeviationBlockPropagation), convert(int(math.Round(float64(result.TimeSimulatedNanos)/result.Stats[0].MeanBlockTime/1000000000))))))

	rewardsResultFile := getOutFile(outDir, fmt.Sprintf("%v%v%v", seed, "_rewards", ".csv"))
	// write header
	_, _ = rewardsResultFile.Write([]byte(fmt.Sprintf("%v ; %v ; %v ; %v ; %v ; %v ; %v\n", "node", "hashrate", "hashrate percentage", "mined blocks", "mined blocks percentage", "rewards", "rewards percentage")))

	for _, hRParent := range result.HashrateAndRewards {
		for _, hR := range hRParent {
			// "seed", "node", "hasrate", "hasrate percentage", "mined blocks", "mined blocks percentage", "rewards", "rewards percentage"
			_, _ = rewardsResultFile.Write([]byte(fmt.Sprintf("%v ; %v ; %v ; %v ; %v ; %v ; %v\n", convert(hR.NodeId), convert(hR.HashRate), convert(hR.HashRatePercentage), convert(hR.MinedBlocks), convert(hR.MinedBlocksPercentage), convert(hR.Rewards), convert(hR.RewardsPercentage))))
		}
		break
	}
}

func convert(v interface{}) string {
	switch v.(type) {
	case float64, float32:
		temp := fmt.Sprintf("%f", v)
		return strings.Replace(temp, ".", ",", -1)
	default:
		return fmt.Sprintf("%v", v)
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

func getOutFile(outPath string, fileName string) *os.File {
	outFile := filepath.Join(outPath, fileName)
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
