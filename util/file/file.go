package file

import (
	"ethattacksim/interfaces"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	CSeed                          uint64          `yaml:"seed"`
	CUseMetrics                    bool            `yaml:"useMetrics"`
	CUsePprof                      bool            `yaml:"usePprof"`
	COutPath                       string          `yaml:"outPath"`
	CPrintLogToConsole             bool            `yaml:"printLogToConsole"`
	CPrintAuditLogToConsole        bool            `yaml:"printAuditLogToConsole"`
	CPrintMemStats                 bool            `yaml:"printMemStats"`
	CEndTime                       int64           `yaml:"endTime"`
	CNodeCount                     uint64          `yaml:"nodeCount"`
	CSimulateTransactionCreation   bool            `yaml:"simulateTransactionCreation"`
	CCheckPastTxWhenVerifyingState bool            `yaml:"checkPastTxWhenVerifyingState"`
	CAuditLogTxMessages            bool            `yaml:"auditLogTxMessages"`
	CNoneNodeUsers                 uint64          `yaml:"noneNodeUsers"`
	CMaxUncleDist                  uint64          `yaml:"maxUncleDist"`
	CTxPerMin                      uint64          `yaml:"txPerMin"`
	CBombDelay                     uint64          `yaml:"bombDelay"`
	COverallHashPower              float64         `yaml:"overallHashPower"`
	CMiningPoolsHashPower          []float64       `yaml:"miningPoolsHashPower"`
	CMiningPoolsCpuPower           []float64       `yaml:"miningPoolsCpuPower"`
	CBlockNephewReward             float64         `yaml:"blockNephewReward"`
	CBlockReward                   float64         `yaml:"blockReward"`
	CLimits                        map[string]int  `yaml:"limits"`
	CSizes                         map[string]int  `yaml:"sizes"`
	CAttackerActive                bool            `yaml:"attackerActive"`
	CAttacker                      *AttackerConfig `yaml:"attacker"`
}

type AttackerConfig struct {
	AType      string             `yaml:"type"`
	AHashPower []float64          `yaml:"hashPower"`
	AMaxPeers  []int              `yaml:"maxPeers"`
	ALocation  []string           `yaml:"location"`
	ACpuPower  []float64          `yaml:"cpuPower"`
	ANumbers   map[string]float64 `yaml:"numbers"`
	AStrings   map[string]string  `yaml:"strings"`
}

func (config *AttackerConfig) Type() string {
	return config.AType
}

func (config *AttackerConfig) HashPower() []float64 {
	return config.AHashPower
}

func (config *AttackerConfig) MaxPeers() []int {
	return config.AMaxPeers
}

func (config *AttackerConfig) CpuPower() []float64 {
	return config.ACpuPower
}

func (config *AttackerConfig) Location() []string {
	return config.ALocation
}

func (config *AttackerConfig) Numbers() map[string]float64 {
	return config.ANumbers
}

func (config *AttackerConfig) Strings() map[string]string {
	return config.AStrings
}

func (config *Config) Seed() uint64 {
	return config.CSeed
}

func (config *Config) UseMetrics() bool {
	return config.CUseMetrics
}

func (config *Config) UsePprof() bool {
	return config.CUsePprof
}

func (config *Config) OutPath() string {
	return config.COutPath
}

func (config *Config) PrintLogToConsole() bool {
	return config.CPrintLogToConsole
}

func (config *Config) PrintAuditLogToConsole() bool {
	return config.CPrintAuditLogToConsole
}

func (config *Config) PrintMemStats() bool {
	return config.CPrintMemStats
}

func (config *Config) EndTime() int64 {
	return config.CEndTime
}

func (config *Config) NodeCount() uint64 {
	return config.CNodeCount
}

func (config *Config) SimulateTransactionCreation() bool {
	return config.CSimulateTransactionCreation
}

func (config *Config) CheckPastTxWhenVerifyingState() bool {
	return config.CCheckPastTxWhenVerifyingState
}

func (config *Config) AuditLogTxMessages() bool {
	return config.CAuditLogTxMessages
}

func (config *Config) NoneNodeUsers() uint64 {
	return config.CNoneNodeUsers
}

func (config *Config) MaxUncleDist() uint64 {
	return config.CMaxUncleDist
}

func (config *Config) TxPerMin() uint64 {
	return config.CTxPerMin
}

func (config *Config) BombDelay() uint64 {
	return config.CBombDelay
}

func (config *Config) OverallHashPower() float64 {
	return config.COverallHashPower
}

func (config *Config) MiningPoolsHashPower() []float64 {
	return config.CMiningPoolsHashPower
}

func (config *Config) MiningPoolsCpuPower() []float64 {
	return config.CMiningPoolsCpuPower
}

func (config *Config) BlockNephewReward() float64 {
	return config.CBlockNephewReward
}

func (config *Config) BlockReward() float64 {
	return config.CBlockReward
}

func (config *Config) Limits() map[string]int {
	return config.CLimits
}

func (config *Config) Sizes() map[string]int {
	return config.CSizes
}

func (config *Config) AttackerActive() bool {
	return config.CAttackerActive
}

func (config *Config) Attacker() interfaces.IAttackerConfig {
	return config.CAttacker
}

type DelaysConfig struct {
	Locations              map[string]map[string]DelayLocationConfig `yaml:"locations"`
	TimeBetweenBlocks      DistributionConfig                        `yaml:"timeBetweenBlocks"` //in s
	TxGas                  DistributionConfig                        `yaml:"txGas"`
	GasPrice               DistributionConfig                        `yaml:"gasPrice"`
	TxStateComputation     float64                                   `yaml:"txStateComputation"`
	BaseHeaderVerification float64                                   `yaml:"baseHeaderVerification"`
	BaseBodyVerification   float64                                   `yaml:"baseBodyVerification"`
	BaseTxVerification     float64                                   `yaml:"baseTxVerification"`
}

type DelayLocationConfig struct {
	Latency           DistributionConfig `yaml:"latency"` // in ms
	SendThroughput    DistributionConfig `yaml:"sendThroughput"`
	ReceiveThroughput DistributionConfig `yaml:"receiveThroughput"`
}

type DistributionConfig struct {
	Distribution string    `yaml:"distribution"`
	Params       []float64 `yaml:"params"`
}

func LoadConfig() *Config {
	var config Config
	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Panic(err)
	}

	return &config
}

func LoadDelaysConfig() *DelaysConfig {
	var config DelaysConfig
	yamlFile, err := ioutil.ReadFile("delays.yml")
	if err != nil {
		log.Panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Panic(err)
	}

	return &config
}

func WorldFile(config *Config) *os.File {
	outFile := fmt.Sprintf("%v/%v/world.json", config.OutPath(), config.Seed())
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func StatsOverviewFile(config *Config) *os.File {
	outFile := fmt.Sprintf("%v/%v/overview.json", config.OutPath(), config.Seed())
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func PprofFile(config interfaces.IConfig, number int) *os.File {
	outFile := fmt.Sprintf("%v/%v/pprof_%v", config.OutPath(), config.Seed(), number)
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func MetricsFile(config *Config) *os.File {
	outFile := fmt.Sprintf("%v/%v/metrics.json", config.OutPath(), config.Seed())
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func LoggerFile(config *Config) *os.File {
	outFile := fmt.Sprintf("%v/%v/log.txt", config.OutPath(), config.Seed())
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func AuditLoggerFile(config *Config) *os.File {
	outFile := fmt.Sprintf("%v/%v/auditLog.csv", config.OutPath(), config.Seed())
	if FileExists(outFile) {
		err := os.Remove(outFile)
		if err != nil {
			log.Panic(err)
		}
	} else {
		EnsureOutPath(fmt.Sprintf("%v/%v", config.OutPath(), config.Seed()))
	}
	outputFile, err := os.Create(outFile)
	if err != nil {
		log.Panic(err)
	}

	return outputFile
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func EnsureOutPath(outPath string) {
	_, err := os.Stat(outPath)
	if os.IsNotExist(err) {
		os.MkdirAll(outPath, os.ModePerm)
	}
}
