package world

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"ethattacksim/util/metrics"
	"fmt"
	"log"
	"runtime"
	"runtime/pprof"
	"time"
)

type World struct {
	queue               interfaces.IQueue
	WTime               int64 `json:"worldTime"` // nanos since sim start
	WStartTime          int64 `json:"startTime"` // unix nanos
	endTime             int64
	WNodes              map[string]interfaces.INode `json:"n"`
	nodeIds             []string
	userIds             []string
	WUsers              map[string]int // map from id to tx nonce
	eventsExecutedCount uint64
	nodeIdCount         uint64
	specialNodeIdCount  map[string]uint64
	blockIdCount        uint64
	userIdCount         uint64
	txIdCount           uint64
	simConfig           interfaces.IConfig
	printMemStats       bool
	simStopped          bool
}

func NewWorld(queue interfaces.IQueue, simConfig interfaces.IConfig) interfaces.IWorld {
	return &World{endTime: simConfig.EndTime(), queue: queue, WTime: 0, WNodes: make(map[string]interfaces.INode), WUsers: make(map[string]int), eventsExecutedCount: 0, nodeIdCount: 0, specialNodeIdCount: make(map[string]uint64), txIdCount: 0, nodeIds: make([]string, 0), userIds: make([]string, 0), userIdCount: 0, simConfig: simConfig, printMemStats: simConfig.PrintMemStats(), simStopped: false}
}

func (world *World) Queue() interfaces.IQueue {
	return world.queue
}

func (world *World) Time() int64 {
	return world.WTime
}

func (world *World) StartTime() int64 {
	return world.WStartTime
}

func (world *World) EndTime() int64 {
	return world.endTime
}

func (world *World) Nodes() map[string]interfaces.INode {
	return world.WNodes
}

func (world *World) Users() map[string]int {
	return world.WUsers
}

func (world *World) AddNodes(nodes ...interfaces.INode) {
	for _, n := range nodes {
		world.WNodes[n.Id()] = n
	}
}

func (world *World) RemoveNode(nodeId string) {
	delete(world.WNodes, nodeId)
}

func (world *World) NewNodeId() string {
	world.nodeIdCount++
	return fmt.Sprintf("node%v", world.nodeIdCount)
}

func (world *World) NewSpecialNodeId(specialId string) string {
	if _, ok := world.specialNodeIdCount[specialId]; !ok {
		world.specialNodeIdCount[specialId] = 0
	}
	world.specialNodeIdCount[specialId]++
	return fmt.Sprintf("node_%v%v", specialId, world.specialNodeIdCount[specialId])
}

func (world *World) NewUserId() string {
	world.userIdCount++
	return fmt.Sprintf("user%v", world.userIdCount)
}

func (world *World) NewBlockHash() string {
	world.blockIdCount++
	return fmt.Sprintf("block%v", world.blockIdCount)
}

func (world *World) NewTxId() string {
	world.txIdCount++
	return fmt.Sprintf("tx%v", world.txIdCount)
}

func (world *World) SimConfig() interfaces.IConfig {
	return world.simConfig
}

func (world *World) NodeIds() []string {
	return world.nodeIds
}

func (world *World) AddNodeIds(ids ...string) {
	world.nodeIds = append(world.nodeIds, ids...)
}

func (world *World) UserIds() []string {
	return world.userIds
}

func (world *World) AddUserIds(ids ...string) {
	world.userIds = append(world.userIds, ids...)
}

func (world *World) StopSim() {
	world.simStopped = true
}

func (world *World) StartSim() {
	world.WStartTime = time.Now().UnixNano()
	log.Printf("Sim startet at real time %v\n", time.Unix(0, world.StartTime()))
	// (while) loop through events until finished
	var ev interfaces.IEvent
	if world.printMemStats {
		fmt.Printf("\tTime \t\t\t\t Events(Queue) \t\t\t\t Heap Alloc GiB \t\t Sys Memory GiB \t NumGarbageCollectionCycles\n")
	}
	for world.Queue().Length() > 0 && world.endTime >= world.Time() && !world.simStopped {
		startTime := time.Now().UnixNano()
		ev = world.Queue().NextEvent()
		metrics.Timer(metrics.NameFormat(interfaces.METRIC_EVENT_REAL_TIME, "FoundEvent_Mus"), time.Duration((time.Now().UnixNano()-startTime)/1000))
		startTime = time.Now().UnixNano()
		if world.endTime >= ev.Time() {
			world.WTime = ev.Time()
			ev.Execute(world)
			world.eventsExecutedCount++
			metrics.Timer(metrics.NameFormat(interfaces.METRIC_EVENT_REAL_TIME, fmt.Sprintf("%v_Mus", ev.Type())), time.Duration((time.Now().UnixNano()-startTime)/1000))
		} else {
			break
		}
		if world.printMemStats && world.eventsExecutedCount%1000 == 0 {
			printMemUsage(false, world, &world.eventsExecutedCount, world.Queue().Length())
		}
		if world.printMemStats && world.Queue().Length() > 50000 && world.eventsExecutedCount%100000 == 0 {
			log.Printf("%v", world.Queue().CountEventTypesAndTimes())
		}
		if world.SimConfig().UsePprof() && world.eventsExecutedCount%1000000 == 0 {
			printPprof(&world.eventsExecutedCount, world.SimConfig())
		}
	}
	if world.printMemStats {
		fmt.Printf("\n")
		printMemUsage(true, world, &world.eventsExecutedCount, world.Queue().Length())
	}
	log.Printf("Sim ended with world time %v (real time %v) after %v (real time %v), %v events were executed\n", world.Time(), time.Unix(0, world.Time()+world.StartTime()), time.Since(time.Unix(0, world.StartTime())), time.Unix(0, world.Time()+world.StartTime()).Sub(time.Unix(0, world.StartTime())), world.eventsExecutedCount)
}

func printMemUsage(toLogger bool, world *World, executedCount *uint64, queueLength int) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if toLogger {
		log.Printf("\n\tHeap Alloc %.3f GiB\n\tTotal (Acc) Heap Alloc %.3f GiB\n\tSys Memory %.3f GiB\n\tNumGarbageCollectionCycles %v\n", bToGb(&m.Alloc), bToGb(&m.TotalAlloc), bToGb(&m.Sys), m.NumGC)
	} else {
		fmt.Printf("\r%20v \t\t %25s \t\t\t %10.3f \t\t\t %10.3f \t\t %10d", time.Unix(0, world.Time()+world.StartTime()).Sub(time.Unix(0, world.StartTime())), fmt.Sprintf("%v(%v)", *executedCount, queueLength), bToGb(&m.Alloc), bToGb(&m.Sys), m.NumGC)
	}
}

// for debugging memory leaks
func printPprof(eventCount *uint64, config interfaces.IConfig) {
	f := file.PprofFile(config, int(*eventCount))
	defer f.Close()
	pprof.WriteHeapProfile(f)
}

func bToGb(b *uint64) float64 {
	return float64(*b) / 1024 / 1024 / 1024
}
