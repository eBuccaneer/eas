package main

import (
	"ethattacksim/util/file"
	"ethattacksim/util/logger"
	"ethattacksim/util/metrics"
	"ethattacksim/util/random"
	"ethattacksim/util/stats"
	"ethattacksim/util/validation"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
)

func main() {
	runs := 1
	if len(os.Args) > 1 {
		parsedRuns, err := strconv.Atoi(os.Args[1])
		if err != nil {
			log.Panic(err)
		}
		runs = parsedRuns
		log.Printf("Sim will be executed %v times\n", runs)
	}

	// load config
	config := file.LoadConfig()
	validation.ValidateConfig(config)

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	initialSeed := config.Seed()
	simShouldStop := false
	for i := initialSeed; i < initialSeed+uint64(runs); i++ {
		if !simShouldStop {
			config.CSeed = i
			// init logger
			loggerFile := file.LoggerFile(config)
			log.SetOutput(logger.NewLogger(loggerFile, config.PrintLogToConsole()))

			// init eventLogger
			auditLoggerFile := file.AuditLoggerFile(config)
			logger.InitAuditLogger(auditLoggerFile, config.PrintAuditLogToConsole())

			// init packages
			random.Initialize(config.Seed())
			metrics.Initialize(config)

			// init world
			simWorld := createWorldAndState(config)

			stopListeningForInterruptChan := make(chan bool, 1)
			go func() {
				for {
					select {
					case <-interruptChan:
						fmt.Println()
						log.Printf("Sim interrupted\n")
						simShouldStop = true
						simWorld.StopSim()
						return
					case <-stopListeningForInterruptChan:
						return
					}
				}
			}()

			// start sim
			simWorld.StartSim()
			stopListeningForInterruptChan <- true

			// write metrics to file if needed
			if config.UseMetrics() {
				f := file.MetricsFile(config)
				metrics.WriteToFile(f)
				_ = f.Close()
			}

			// print stats to file
			stats.PrintWorld(simWorld, file.WorldFile(config))
			stats.PrintStatsOverview(simWorld, file.StatsOverviewFile(config), config)

			// just for testing of determinism
			random.PrintCount()
			random.PrintDelaysCount()

			_ = loggerFile.Close()
			_ = auditLoggerFile.Close()
		}
	}
}
