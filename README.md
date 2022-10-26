# ethattacksim

This is an Ethereum blockchain simulator.
The simulator offers various attack simulations too and can be easily extended.
Additionally, there are multiple scripts available in the scripts directory:
- analyze: analyzing of the output of simulator runs
- blockData: a script to retrieve Ethereum blockchain data
- fitDistribution: a script to fit output data to the best-suiting distribution
- summarize: to summarize data created by the analyze script

## Build Binary
Run `build.sh` from root directory

The binaries for the go scripts are created by the `build.sh` scripts in the respective script directories

## Run from Source
`go run *.go` from `ethattacksim/main` directory or from the respective script directories

## Program arguments
### ethattacksim
`./ethattacksim[.exe] [RUNS]` ... integer that indicates the number of runs the simulator should execute
- `[RUNS]` ... integer that indicates the number of runs the simulator should execute

### analyze
`./analyze[.exe] [NO_EIP1559 | EIP1559] [IN_DIR] [OUT_DIR]`
- `NO_EIP1559 | EIP1559` ... indicates if transaction fees should be calculated according to EIP1559 or not, default is pre-EIP1559
- `IN_DIR` ... the input directory that simulator output is taken from, defaults to `../../out`
- `OUT_DIR` .. the output directory, defaults to `./out`

### fitDistribution
`./fitDistribution[.exe] FILE_PATH|DIR_PATH [OUT_DIR] [SAMPLES_COUNT] [SEED]`
- `FILE_PATH|DIR_PATH` ... the input path to the .txt file that should be fit
- `OUT_DIR` ... the output directory
- `SAMPLES_COUNT` ... integer indicating the amount of samples drawn from each distribution to check
- `SEED` ... an integer seed for randomization of distribution checks

### summarize
`./summarize[.exe] [IN_DIR] [OUT_DIR]`
- `IN_DIR` ... the input directory that analyze script output is taken from, defaults to `../analyze/out`
- `OUT_DIR` .. the output directory, defaults to `./out`

## Memory Inspection
- start `go tool pprof [PPROF_FILE_PATH]`
- `top` to show top consumers
- `png` to save a visualization

## Useful Commands
execute summarize script for all folders in current directory
`find ~+ -mindepth 1 -maxdepth 1 -type d \( ! -name summary \) -exec bash -c "/path/to/summarize/bin/summarize-darwin-arm64 {} {}/summary" \;`

## Notes
- indeterminism with same seed can happen occasionally, if TXs are propagated through the network (i.e. when TX creation is active), because ordering is not deterministic if i.e. same gasPrice is used
- indeterminism can also happen because per definition if the max blocks/tx monitored (if peers have seen it) limit is reached, an arbitrary entry gets deleted

## Data
Data used to fit distributions and retrieve certain important parameters for simulation is contained in `/data` directory, together with the results of distribution fitting.

### Credits
Data contained in `/data/blocksim_faria` that was used to create ping and throughput distributions was taken from the paper of Carlos Faria and Miguel Correia and the respective thesis of Carlos Faria:
BlockSim: Blockchain Simulator. In 2019 IEEE International Conference on Blockchain (Blockchain)
URL: `https://ieeexplore.ieee.org/document/8946201/`
