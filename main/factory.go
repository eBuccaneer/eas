package main

import (
	"ethattacksim/consensus"
	attackConsensus "ethattacksim/consensus/attack"
	"ethattacksim/event"
	"ethattacksim/event/events"
	"ethattacksim/interfaces"
	"ethattacksim/ledger"
	"ethattacksim/network"
	"ethattacksim/node"
	"ethattacksim/util/file"
	"ethattacksim/util/random"
	"ethattacksim/world"
	"log"
	"math"
	"sort"
)

func createWorldAndState(config *file.Config) interfaces.IWorld {

	// create new event queue
	queue := event.NewQueue()
	var simWorld interfaces.IWorld = world.NewWorld(queue, config)

	delaysConfig := file.LoadDelaysConfig()
	random.InitializeDelays(config.Seed(), delaysConfig)

	var freePower float64 = config.OverallHashPower()
	var location interfaces.ILocation

	// init mining pools
	for i, poolPower := range config.MiningPoolsHashPower() {
		location = locationOracle()
		poolCpuPower := config.MiningPoolsCpuPower()[i]
		simWorld.AddNodes(node.NewNode(simWorld.NewSpecialNodeId("pool"), poolPower, poolCpuPower, interfaces.FULL_NODE, location, ledger.NewLedger(), network.NewNetwork(poolPeerCountOracle()), consensus.NewConsensus()))
		freePower -= poolPower
	}
	poolsPower := config.OverallHashPower() - freePower

	if config.AttackerActive() {
		var attackerNodeIds []string = make([]string, 0, len(simWorld.Nodes()))
		for i, attackerNodePower := range config.Attacker().HashPower() {
			attackerNodeCpuPower := config.Attacker().CpuPower()[i]
			attackerNodeMaxPeers := config.Attacker().MaxPeers()[i]
			attackerNodeLocation := interfaces.LOCATION_MAP[config.Attacker().Location()[i]]
			attackerNodeId := simWorld.NewSpecialNodeId("attacker")
			attackerNodeIds = append(attackerNodeIds, attackerNodeId)
			switch config.Attacker().Type() {
			// create special nodes according to attacker type here
			case "verifiersDilemma":
				simWorld.AddNodes(node.NewNode(attackerNodeId, attackerNodePower, attackerNodeCpuPower, interfaces.ATTACKER_NODE, attackerNodeLocation, ledger.NewLedger(), network.NewNetwork(attackerNodeMaxPeers), attackConsensus.NewVerifiersDilemmaConsensus(consensus.NewConsensus())))
				break
			case "verifiersDilemmaForced":
				simWorld.AddNodes(node.NewNode(attackerNodeId, attackerNodePower, attackerNodeCpuPower, interfaces.ATTACKER_NODE, attackerNodeLocation, ledger.NewLedger(), network.NewNetwork(attackerNodeMaxPeers), attackConsensus.NewVerifiersDilemmaConsensusForced(consensus.NewConsensus())))
				break
			case "selfishMining":
				simWorld.AddNodes(node.NewNode(attackerNodeId, attackerNodePower, attackerNodeCpuPower, interfaces.ATTACKER_NODE, attackerNodeLocation, ledger.NewLedger(), network.NewNetwork(attackerNodeMaxPeers), attackConsensus.NewSelfishMiningConsensus(consensus.NewConsensus())))
				break
			default:
				simWorld.AddNodes(node.NewNode(attackerNodeId, attackerNodePower, attackerNodeCpuPower, interfaces.ATTACKER_NODE, attackerNodeLocation, ledger.NewLedger(), network.NewNetwork(attackerNodeMaxPeers), attackConsensus.NewVerifiersDilemmaConsensus(consensus.NewConsensus())))
			}
			freePower -= attackerNodePower
		}

		// peer attacker nodes all together
		// use sorted node key array because of determinism
		sort.Strings(attackerNodeIds)
		for _, nId := range attackerNodeIds {
			for _, nId2 := range attackerNodeIds {
				if nId != nId2 {
					n1 := simWorld.Nodes()[nId]
					n2 := simWorld.Nodes()[nId2]
					if !containsPeer(n1, n2) {
						n1.AddPeers(n2)
					}
					if !containsPeer(n2, n1) {
						n2.AddPeers(n1)
					}
				}
			}
		}
	}

	attackerNodesInitialized := len(config.Attacker().HashPower())
	attackerPower := config.OverallHashPower() - poolsPower - freePower
	if !config.AttackerActive() {
		attackerNodesInitialized = 0
	}

	// init other nodes
	remainingNodes := int(config.NodeCount()) - len(config.MiningPoolsHashPower()) - attackerNodesInitialized
	avg := freePower / float64(remainingNodes)

	log.Printf("created %v pools with %v TH, %v attackers with %v TH power, distributing %v TH (avg %v TH) to %v other nodes\n", len(config.MiningPoolsHashPower()), poolsPower/1000000, attackerNodesInitialized, attackerPower/1000000, freePower/1000000, avg/1000000, remainingNodes)

	for i := 0; i < int(config.NodeCount())-len(config.MiningPoolsHashPower())-attackerNodesInitialized; i++ {
		location = locationOracle()
		power := hashPowerOracle(avg, freePower, remainingNodes)
		simWorld.AddNodes(node.NewNode(simWorld.NewNodeId(), power, cpuPowerOracle(), interfaces.FULL_NODE, location, ledger.NewLedger(), network.NewNetwork(peerCountOracle()), consensus.NewConsensus()))
		remainingNodes--
		freePower -= power
	}

	// use sorted node key array because of determinism
	var nodeIds []string = make([]string, 0, len(simWorld.Nodes()))
	for nId, _ := range simWorld.Nodes() {
		nodeIds = append(nodeIds, nId)
	}
	sort.Strings(nodeIds)
	simWorld.AddNodeIds(nodeIds...)

	// init peers
	for _, nId := range nodeIds {
		simWorld.Nodes()[nId].Network().ConnectToPeers(nId, simWorld)
	}

	// init non-node users
	for i := 0; i < int(config.NoneNodeUsers()); i++ {
		simWorld.Users()[simWorld.NewUserId()] = 0
	}
	// use sorted node key array because of determinism
	var userIds []string = make([]string, 0, config.NoneNodeUsers())
	for uId, _ := range simWorld.Users() {
		userIds = append(userIds, uId)
	}
	sort.Strings(userIds)
	simWorld.AddUserIds(userIds...)

	queue.SetWorld(simWorld) // assigns world to queue and inits initial queue size

	// init genesis event
	var genHeader interfaces.IBlockHeader
	for _, nId := range nodeIds {
		n := simWorld.Nodes()[nId]
		genHeader = ledger.NewBlockHeader("GENESIS", "GENESIS", "", "", "GENESIS", 8, -1, simWorld.SimConfig().Limits()["initialGasLimit"], 0, 0, 0, true)
		body := ledger.NewBlockBody(genHeader.Hash(), make([]interfaces.ITransaction, 0), make([]interfaces.IBlockHeader, 0), true, 0)
		block := ledger.NewBlock(genHeader, body, 8)
		queue.Add(events.NewGenesisEvent(event.NewEvent(0, n.Id(), interfaces.GENESIS_EVENT), block))
	}

	if config.SimulateTransactionCreation() {
		// init first tx creation event
		queue.Add(events.NewTxCreationEvent(event.NewEvent(0, "WORLD", interfaces.TX_CREATION_EVENT)))
	}

	log.Print("init complete")
	return simWorld
}

func cpuPowerOracle() float64 {
	return math.Max(3.3+random.Uniform()*1.2, 3.3) * 1000 // 3.3 - 4.5 GHz
}

// between 25 and 50 peers per pool
func poolPeerCountOracle() int {
	return int(math.Max(25+random.Uniform()*26, 25))
}

// between 15 and 25 peers per node
func peerCountOracle() int {
	return int(math.Max(15+random.Uniform()*11, 15))
}

func hashPowerOracle(avg float64, remaining float64, remainingCount int) float64 {
	if remainingCount == 1 {
		return remaining
	}
	minMH := 1000.0
	maxTimesAvg := 2.5
	rand := random.Normal()
	if rand < -1 {
		rand /= 2 // to minimize the values below -1
	}
	plusMinus := math.Min(math.Max(rand, -1), maxTimesAvg-1) * avg
	power := math.Max(plusMinus+avg, minMH)
	correctedPower := math.Min(power, remaining-(float64(remainingCount-1)*minMH)) // to have enough power left for the remaining nodes
	return math.Max(correctedPower, 1)                                             // to prevent negative hashpower
}

func locationOracle() interfaces.ILocation {
	switch int(random.Uniform() * 3) {
	case 0:
		return interfaces.TOKIO
	case 1:
		return interfaces.IRELAND
	case 2:
		return interfaces.OHIO
	default:
		return interfaces.TOKIO
	}
}

func containsPeer(n1 interfaces.INode, n2 interfaces.INode) bool {
	for _, p := range n1.Peers() {
		if p.Id() == n2.Id() {
			return true
		}
	}
	return false
}
