package main

import (
	"bufio"
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"ethattacksim/util/logger"
	"ethattacksim/util/random"
	"fmt"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		panic("Please use format './fitDistribution[.exe] FILE_PATH|DIR_PATH [OUT_DIR] [SAMPLES_COUNT:int] [SEED:int]' on command line.")
	}
	fileName := os.Args[1] // "time_blocks_eth.txt"
	var samplesCount int
	var err error

	outDir := "./out"
	if len(os.Args) >= 3 {
		outDir = os.Args[2]
		log.Printf("Using outDir '%v'.", outDir)
	} else {
		log.Printf("Using standard output dir '%v'.", outDir)
	}

	if len(os.Args) >= 4 {
		samplesCount, err = strconv.Atoi(os.Args[3]) // amount of samples drawn at each distribution check
	} else {
		samplesCount = 100
	}

	var seed int
	if len(os.Args) >= 5 {
		seed, err = strconv.Atoi(os.Args[4]) // seed of random utils
	} else {
		seed = 0
	}

	if err != nil {
		panic(err)
	}

	isFile := checkIsFile(fileName)

	if isFile {
		processSingle(fileName, samplesCount, seed, outDir)
	} else {
		process(fileName, samplesCount, seed, outDir)
	}
}

type Result struct {
	BestF      float64
	BestParams []float64
	Name       string
}

type Job struct {
	Name    string
	Problem optimize.Problem
	Min     float64
	Max     float64
	Factor  float64
	Min2    float64
	Max2    float64
	Factor2 float64
}

func process(dirName string, samplesCount int, seed int, outPath string) {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		panic("No files to process in directory " + dirName)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".txt") {
			processSingle(filepath.Join(dirName, f.Name()), samplesCount, seed, outPath)
		}
	}
}

func processSingle(fileName string, samplesCount int, seed int, outPath string) {
	// init
	loggerFile := getOutFile("result_", fileName, outPath)
	defer loggerFile.Close()
	log.SetOutput(logger.NewLogger(loggerFile, true))

	inFile := loadFile(fileName)
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	var values []float64 = make([]float64, 0, 10)
	var valuesWeight []float64 = make([]float64, 0, 10)
	if strings.Contains(fileName, "ping") {
		// read from txt file (containing ping output)
		for scanner.Scan() {
			text := scanner.Text()
			index := strings.Index(text, "time=")
			text = text[index+5:]
			index = strings.Index(text, " ")
			text = text[:index]
			val, _ := strconv.ParseFloat(text, 64)
			values = append(values, val)
			valuesWeight = append(valuesWeight, 1)
		}
	} else if strings.Contains(fileName, "sent") || strings.Contains(fileName, "received") || strings.Contains(fileName, "time") {
		// read from txt file (containing only floats)
		for scanner.Scan() {
			text := scanner.Text()
			val, _ := strconv.ParseFloat(text, 64)
			values = append(values, val)
			valuesWeight = append(valuesWeight, 1)
		}
	} else if strings.Contains(fileName, "blockPropagation") || strings.Contains(fileName, "reception") || strings.Contains(fileName, "txGasPrice") {
		// read from txt file (containing only ints)
		for scanner.Scan() {
			text := scanner.Text()
			val, _ := strconv.ParseInt(text, 10, 64)
			values = append(values, float64(val))
			valuesWeight = append(valuesWeight, 1)
		}
	} else if strings.Contains(fileName, "txGas") {
		// read from txt file (containing only ints)
		for scanner.Scan() {
			text := scanner.Text()
			val, _ := strconv.ParseInt(text, 10, 64)
			values = append(values, float64(val)-21000) // as 21000 is the min value, we are only interested in the rest
			valuesWeight = append(valuesWeight, 1)
		}
	}
	log.Printf("read %v values from file %v\n", len(values), fileName)
	sort.Float64s(values)

	minObserved := values[0]
	maxObserved := values[len(values)-1]

	resultC := make(chan *Result, 12)
	workerC := make(chan *Job)

	// compute results
	fmt.Printf("Starting %v workers...\n", runtime.GOMAXPROCS(0))
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go doWork(i, workerC, resultC)
	}

	workerC <- &Job{"beta", getProblem("beta", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}
	workerC <- &Job{"invgamma", getProblem("invgamma", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}
	workerC <- &Job{"norm", getProblem("norm", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, false), minObserved, maxObserved, 1.5, 0, maxObserved, 1.5}
	workerC <- &Job{"gamma", getProblem("gamma", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}
	workerC <- &Job{"lognorm", getProblem("lognorm", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, false), -100, maxObserved, 1.5, 0, maxObserved, 1.5}
	workerC <- &Job{"chisquare", getProblem("chisquare", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, -1, -1, 1.5}
	workerC <- &Job{"exp", getProblem("exp", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, false), -100, 100, 1.5, -1, -1, 1.5}
	workerC <- &Job{"F", getProblem("F", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}
	workerC <- &Job{"laplace", getProblem("laplace", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, false), minObserved, maxObserved, 1.5, minObserved, maxObserved, 1.5}
	workerC <- &Job{"pareto", getProblem("pareto", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}
	workerC <- &Job{"uniform", getProblem("uniform", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, false), minObserved, maxObserved, 1.5, minObserved, maxObserved, 1.5}
	workerC <- &Job{"weibull", getProblem("weibull", rand.NewSource(uint64(seed)), values, valuesWeight, samplesCount, true), 0.01, 1000000, 1.5, 0.01, 1000000, 1.5}

	close(workerC)

	var results map[string]*Result = make(map[string]*Result)
	for i := 0; i < 12; i++ {
		res := <-resultC
		results[res.Name] = res
	}

	// find and print result
	fmt.Print("\n")
	var bestResult []float64
	var bestF float64 = math.MaxFloat64
	var bestDist string
	for k, v := range results {
		log.Printf("%v: F %v, params %v\n", k, v.BestF, v.BestParams)
		if v.BestF < bestF {
			bestF = v.BestF
			bestResult = v.BestParams
			bestDist = k
		}
	}
	log.Printf("best suiting distribution is %v with F %v and parameters %v\n", bestDist, bestF, bestResult)
	printSamplesOfWinner(bestDist, bestResult, 50, rand.NewSource(uint64(seed)))
}

func doWork(n int, workerC <-chan *Job, resultC chan<- *Result) {
	for {
		job, more := <-workerC
		if more {
			getBestResult(job.Name, resultC, job.Problem, job.Min, job.Max, job.Factor, job.Min2, job.Max2, job.Factor2)
		} else {
			return
		}
	}
}

func printSamplesOfWinner(winnerDist string, params []float64, n int, source rand.Source) {
	var rng interfaces.IRNG = random.GetDist(winnerDist, params, source)
	var result []float64 = make([]float64, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, rng.Rand())
	}
	log.Printf("samples from winner:\n %v\n", result)
}

func getProblem(distName string, source rand.Source, values []float64, valuesWeight []float64, samplesCount int, xBigger0 bool) optimize.Problem {
	return optimize.Problem{
		Func: func(x []float64) float64 {
			if len(x) > 1 {
				if xBigger0 && (x[0] <= 0 || x[1] <= 0) {
					return math.MaxFloat64
				}
			} else {
				if xBigger0 && (x[0] <= 0) {
					return math.MaxFloat64
				}
			}
			randFunc := random.GetDist(distName, x, source)
			var samples []float64 = make([]float64, 0, samplesCount)
			var samplesWeight []float64 = make([]float64, 0, samplesCount)
			for i := 0; i < samplesCount; i++ {
				samples = append(samples, randFunc.Rand())
				samplesWeight = append(samplesWeight, 1)
			}
			sort.Float64s(samples)
			ks := stat.KolmogorovSmirnov(values, valuesWeight, samples, samplesWeight)
			if math.IsNaN(ks) {
				ks = math.MaxFloat64
			}
			return ks
		},
	}
}

func getBestResult(name string, resultsC chan<- *Result, problem optimize.Problem, min float64, max float64, factor float64, min2 float64, max2 float64, factor2 float64) {
	var bestResult []float64
	var bestF float64 = math.MaxFloat64
	i := min
	for i <= max {
		j := min2
		for j <= max2 {
			var result *optimize.Result
			var err error
			if min2 == max2 && max2 == -1 {
				// here we only have one distribution parameter
				result, err = optimize.Minimize(problem, []float64{i}, &optimize.Settings{Concurrent: 4, Converger: &optimize.FunctionConverge{Absolute: 1e-15, Iterations: 100}}, &optimize.NelderMead{})
			} else {
				result, err = optimize.Minimize(problem, []float64{i, j}, &optimize.Settings{Concurrent: 4, Converger: &optimize.FunctionConverge{Absolute: 1e-15, Iterations: 100}}, &optimize.NelderMead{})
			}
			if err != nil {
				log.Printf("%v, %v, %v, %v", min, max, min2, max2)
				log.Panic(err)
			}
			if result.F < bestF {
				bestF = result.F
				bestResult = result.X
			}

			// adapt j
			if j < 0 {
				j /= factor2
			}
			if j > 0 {
				j *= factor2
			}
			if j < 0 && j > -0.01 {
				j = math.Abs(j)
			}
			if j == 0 {
				j += 0.01
			}
		}

		// adapt i
		if i < 0 {
			i /= factor
		}
		if i > 0 {
			i *= factor
		}
		if i < 0 && i > -0.01 {
			i = math.Abs(i)
		}
		if i == 0 {
			i += 0.01
		}
	}
	fmt.Print(".")
	resultsC <- &Result{bestF, bestResult, name}
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

func getOutFile(prefix string, name string, outPath string) *os.File {
	_, fileName := filepath.Split(name)
	//outFile := outPath + "/" + prefix + name
	outFile := filepath.Join(outPath, prefix+fileName)
	if file.FileExists("outFile") {
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
