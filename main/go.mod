module ethattacksim/main

go 1.13

require (
	ethattacksim/consensus v0.0.0
	ethattacksim/event v0.0.0
	ethattacksim/interfaces v0.0.0
	ethattacksim/ledger v0.0.0
	ethattacksim/network v0.0.0
	ethattacksim/node v0.0.0
	ethattacksim/util v0.0.0
	ethattacksim/world v0.0.0
)

replace ethattacksim/world => ../world

replace ethattacksim/consensus => ../consensus

replace ethattacksim/network => ../network

replace ethattacksim/event => ../event

replace ethattacksim/interfaces => ../interfaces

replace ethattacksim/util => ../util

replace ethattacksim/node => ../node

replace ethattacksim/ledger => ../ledger
