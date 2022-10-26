module fitDistribution/main

go 1.13

require (
	ethattacksim/interfaces v0.0.0
	ethattacksim/util v0.0.0
	ethattacksim/world v0.0.0
	golang.org/x/exp v0.0.0-20200513190911-00229845015e
	gonum.org/v1/gonum v0.7.0
)

replace ethattacksim/util => ../../util

replace ethattacksim/world => ../../world

replace ethattacksim/interfaces => ../../interfaces
