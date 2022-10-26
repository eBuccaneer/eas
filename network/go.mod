module ethattacksim/network

go 1.13

require (
	ethattacksim/event v0.0.0
	ethattacksim/interfaces v0.0.0
	ethattacksim/ledger v0.0.0
	ethattacksim/util v0.0.0
)

replace ethattacksim/event => ../event

replace ethattacksim/util => ../util

replace ethattacksim/ledger => ../ledger

replace ethattacksim/interfaces => ../interfaces
