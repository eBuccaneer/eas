package random

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"golang.org/x/exp/rand"
	"log"
	"math"
)

//var timeBetweenBlocks interfaces.IRNG
var txGas interfaces.IRNG
var gasPrice interfaces.IRNG
var timeBetweenBlocksSource rand.Source
var cfg *file.DelaysConfig

var delaysRNGMap map[interfaces.ILocation]map[interfaces.ILocation]*DelaysRNG

var txValidationCount int
var timeBetweenBlocksCount int
var txGasCount int
var delaysMapCount int

func TxStateComputation(gas int, cpuPower float64, specialTxStateComputation float64) int64 {
	txValidationCount++
	if specialTxStateComputation > -1 {
		return int64(float64(gas) / (specialTxStateComputation * cpuPower) * 1000000000)
	}
	return int64(float64(gas) / (cfg.TxStateComputation * cpuPower) * 1000000000)
}

func BaseHeaderVerification(hashPower float64, cpuPower float64) int64 {
	base := int64(1 / (cfg.BaseHeaderVerification * cpuPower) * 1000000000)
	return HashComputation(hashPower, 1) + base
}

func BaseBodyVerification(hashPower float64, cpuPower float64) int64 {
	base := int64(1 / (cfg.BaseBodyVerification * cpuPower) * 1000000000)
	return HashComputation(hashPower, 2) + base
}

func BaseTxVerification(hashPower float64, cpuPower float64) int64 {
	base := int64(1 / (cfg.BaseTxVerification * cpuPower) * 1000000000)
	return HashComputation(hashPower, 1) + base
}

func HashComputation(hashPower float64, n int) int64 {
	return int64(math.Max((1/(hashPower*1000000))*1000000000*float64(n), 1)) // at least 1 ns
}

// blockTimeStampDelay == miningTimeDelay iff nodeTime + blockTimeStampDelay >= lastBlockTime + 1s; otherwise blockTimeStampDelay == lastBlockTime + 1s - nodeTime
func TimeBetweenBlocks(totalHashPower float64, hashPower float64, lastBlockTime int64, nodeTime int64) (blockTimeStamp int64, miningTimeDelay int64) {
	timeBetweenBlocksCount++
	power := hashPower / totalHashPower
	// timeBetweenBlocks uses exponential distribution with lambda = fraction of hashpower * (1 / targeted mean block time)
	timeBetweenBlocks := GetDist(cfg.TimeBetweenBlocks.Distribution, []float64{power * (1 / cfg.TimeBetweenBlocks.Params[0])}, timeBetweenBlocksSource)
	blockTimeStampDelay := int64(math.Round(timeBetweenBlocks.Rand() * 1000000000))
	if blockTimeStampDelay < 0 || blockTimeStampDelay > 86400000000000 { // if overflow or bigger than one day (should be big enough)
		blockTimeStampDelay = 86400000000000
	}
	miningTimeDelay = blockTimeStampDelay
	minNewBlockTimeStamp := lastBlockTime + 1                             // block time is in seconds
	if (nodeTime+blockTimeStampDelay)/1000000000 < minNewBlockTimeStamp { // timestamp must be at least one second higher than last, even if block mined slightly earlier
		blockTimeStamp = minNewBlockTimeStamp
	} else {
		blockTimeStamp = (nodeTime + blockTimeStampDelay) / 1000000000
	}
	return
}

func TxGas(min int) int64 {
	txGasCount++
	gas := int64(math.Round(txGas.Rand()))
	return int64(math.Max(float64(gas), 0)) + int64(min)
}

func GasPrice() int64 {
	price := int64(math.Round(gasPrice.Rand()))
	if price < 1 {
		price = 1
	}
	return price
}

func Latency(origin interfaces.ILocation, destination interfaces.ILocation) int64 {
	delaysMapCount++
	if _, ok := delaysRNGMap[origin]; !ok {
		log.Panic("latency of origin " + origin.String() + " not in map")
	}
	if _, ok := delaysRNGMap[origin][destination]; !ok {
		log.Panic("latency of origin " + origin.String() + " and destination " + destination.String() + " not in map")
	}
	val := int64(delaysRNGMap[origin][destination].Latency.Rand() * 1000000)
	if val <= 0 {
		return Latency(origin, destination)
	}
	return val
}

func ReceiveThroughput(origin interfaces.ILocation, destination interfaces.ILocation, bytes int) int64 {
	delaysMapCount++
	if _, ok := delaysRNGMap[origin]; !ok {
		log.Panic("receivedThroughput of origin " + origin.String() + " not in map")
	}
	if _, ok := delaysRNGMap[origin][destination]; !ok {
		log.Panic("receivedThroughput of origin " + origin.String() + " and destination " + destination.String() + " not in map")
	}
	mbps := delaysRNGMap[origin][destination].ReceiveThroughput.Rand()
	mbit := float64(bytes) * 8 / 1000000
	val := int64((mbit / mbps) * 1000000000)
	if val <= 0 {
		return ReceiveThroughput(origin, destination, bytes)
	}
	return val
}

func SendThroughput(origin interfaces.ILocation, destination interfaces.ILocation, bytes int) int64 {
	delaysMapCount++
	if _, ok := delaysRNGMap[origin]; !ok {
		log.Panic("sentThroughput of origin " + origin.String() + " not in map")
	}
	if _, ok := delaysRNGMap[origin][destination]; !ok {
		log.Panic("sentThroughput of origin " + origin.String() + " and destination " + destination.String() + " not in map")
	}
	mbps := delaysRNGMap[origin][destination].SendThroughput.Rand()
	mbit := float64(bytes) * 8 / 1000000
	val := int64((mbit / mbps) * 1000000000)
	if val <= 0 {
		return SendThroughput(origin, destination, bytes)
	}
	return val
}

func PrintDelaysCount() {
	log.Printf("random number generators delays call count (indicates determinism) -> txValidation: %v, timeBetweenBlocks: %v, txGas: %v, delaysMap: %v", txValidationCount, timeBetweenBlocksCount, txGasCount, delaysMapCount)
}

// must be called before usage
func InitializeDelays(seed uint64, config *file.DelaysConfig) {
	txValidationCount = 0
	timeBetweenBlocksCount = 0
	txGasCount = 0
	delaysMapCount = 0
	cfg = config

	/*var timeBetweenBlocksSource rand.Source = rand.NewSource(seed)
	timeBetweenBlocks = GetDist(config.TimeBetweenBlocks.Distribution, config.TimeBetweenBlocks.Params, timeBetweenBlocksSource)*/
	timeBetweenBlocksSource = rand.NewSource(seed)

	var txGasSource rand.Source = rand.NewSource(seed)
	txGas = GetDist(config.TxGas.Distribution, config.TxGas.Params, txGasSource)

	var gasPriceSource rand.Source = rand.NewSource(seed)
	gasPrice = GetDist(config.GasPrice.Distribution, config.GasPrice.Params, gasPriceSource)

	delaysRNGMap = make(map[interfaces.ILocation]map[interfaces.ILocation]*DelaysRNG)
	for originKey, destinationMap := range config.Locations {
		for destinationKey, delaysConfig := range destinationMap {
			origin := interfaces.LOCATION_MAP[originKey]
			destination := interfaces.LOCATION_MAP[destinationKey]
			latencyRng := getRNGFromDistributionConfig(seed, &delaysConfig.Latency)
			sendThroughputRng := getRNGFromDistributionConfig(seed, &delaysConfig.SendThroughput)
			receiveThroughputRng := getRNGFromDistributionConfig(seed, &delaysConfig.ReceiveThroughput)
			if delaysRNGMap[origin] == nil {
				delaysRNGMap[origin] = make(map[interfaces.ILocation]*DelaysRNG)
			}
			delaysRNGMap[origin][destination] = &DelaysRNG{latencyRng, sendThroughputRng, receiveThroughputRng}
		}
	}
}

func getRNGFromDistributionConfig(seed uint64, config *file.DistributionConfig) interfaces.IRNG {
	return GetDist(config.Distribution, config.Params, rand.NewSource(seed))
}

type DelaysRNG struct {
	Latency           interfaces.IRNG
	SendThroughput    interfaces.IRNG
	ReceiveThroughput interfaces.IRNG
}
