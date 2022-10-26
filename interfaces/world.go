package interfaces

type IWorld interface {
	Queue() IQueue
	Time() int64      //nanos since start
	StartTime() int64 //unix nanos
	EndTime() int64
	StartSim()
	StopSim()
	Nodes() map[string]INode
	NodeIds() []string
	AddNodeIds(ids ...string)
	Users() map[string]int
	UserIds() []string
	AddUserIds(ids ...string)
	AddNodes(nodes ...INode)
	RemoveNode(nodeId string)
	NewNodeId() string
	NewSpecialNodeId(specialId string) string
	NewUserId() string
	NewBlockHash() string
	NewTxId() string
	SimConfig() IConfig
}
