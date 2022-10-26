package events

import (
	"ethattacksim/event"
	"ethattacksim/interfaces"
	"ethattacksim/ledger"
	"ethattacksim/util/metrics"
	"ethattacksim/util/random"
	"fmt"
	ti "time"
)

/**
event that mimics the network (some nodes) creating transactions
*/
type TxCreationEvent struct {
	interfaces.IEvent
}

func NewTxCreationEvent(ev interfaces.IEvent) *TxCreationEvent {
	return &TxCreationEvent{ev}
}

func (ev *TxCreationEvent) Execute(world interfaces.IWorld) {
	txPerMin := world.SimConfig().TxPerMin()

	for i := 0; i < int(txPerMin); i++ {
		randTime := int64(random.Uniform() * 60000000000) // tx will be created in the next 60 seconde
		randSenderId, senderNonce := txSenderOracle(world.Users(), world.UserIds())
		txGas := int(random.TxGas(world.SimConfig().Limits()["minTxGas"]))
		if txGas > world.SimConfig().Limits()["initialGasLimit"]-500000 {
			// the subtraction is for not having to track current gas limit,
			// so the biggest tx that is automatically created will be initialGasLimit - 500000
			txGas = world.SimConfig().Limits()["initialGasLimit"] - 500000
		}

		gasPrice := int(random.GasPrice())
		specialTxStateComputation := -1.0 // stays at -1 (= not used) for honest nodes

		randTarget := nodeOracle(world.Nodes(), world.NodeIds()).Id()
		tx := ledger.NewTx(fmt.Sprintf("%v_%v", randSenderId, senderNonce), senderNonce, randSenderId, txGas, gasPrice, true, specialTxStateComputation)
		world.Queue().Add(NewNewTxEvent(event.NewEvent(ev.Time()+randTime, randTarget, interfaces.RECEIVED_TXS_EVENT), tx, randSenderId))
	}
	// fire event every minute
	world.Queue().Add(NewTxCreationEvent(event.NewEvent(ev.Time()+60000000000, "", interfaces.NEW_TX_EVENT)))
}

func CreateRandomTransactionsForBlock(gasUsed int, world interfaces.IWorld, gasLimit int) ([]interfaces.ITransaction, int) {
	txs := make([]interfaces.ITransaction, 0)
	gas := 0

	for gas+gasUsed < gasLimit {
		randSenderId, senderNonce := txSenderOracle(world.Users(), world.UserIds())
		txGas := int(random.TxGas(world.SimConfig().Limits()["minTxGas"]))
		if gas+gasUsed+txGas > gasLimit {
			break
		}
		if txGas > world.SimConfig().Limits()["initialGasLimit"]-500000 {
			// the subtraction is for not having to track current gas limit,
			// so the biggest tx that is automatically created will be initialGasLimit - 500000
			txGas = world.SimConfig().Limits()["initialGasLimit"] - 500000
		}

		gasPrice := int(random.GasPrice())
		specialTxStateComputation := -1.0 // stays at -1 (= not used) for honest nodes

		tx := ledger.NewTx(fmt.Sprintf("R%v_%v", randSenderId, senderNonce), senderNonce, randSenderId, txGas, gasPrice, true, specialTxStateComputation)
		txs = append(txs, tx)
		gas += txGas
	}
	for _, t := range txs {
		metrics.Timer(interfaces.METRIC_TX_GAS.String(), ti.Duration(t.GasUsed()))
		metrics.Timer(interfaces.METRIC_TX_PRICE.String(), ti.Duration(t.GasPrice()))
	}
	return txs, gas
}

func nodeOracle(nodes map[string]interfaces.INode, keySet []string) (selectedNode interfaces.INode) {
	selectedNode = nil
	for selectedNode == nil {
		i := int(random.Uniform() * float64(len(keySet)))
		selectedNode = nodes[keySet[i]]
	}
	return
}

func txSenderOracle(users map[string]int, userKeySet []string) (selectedSenderId string, nonce int) {
	// TX by default are not sent by mining nodes, only by users
	i := int(random.Uniform() * float64(len(userKeySet)))
	selectedSenderId = userKeySet[i]
	nonce = users[userKeySet[i]]
	users[userKeySet[i]] += 1
	return
}
