module ethattacksim/util

go 1.13

require (
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	ethattacksim/interfaces v0.0.0
	golang.org/x/exp v0.0.0-20200513190911-00229845015e
	gonum.org/v1/gonum v0.7.0
	gopkg.in/yaml.v2 v2.2.4
)

replace ethattacksim/interfaces => ../interfaces
