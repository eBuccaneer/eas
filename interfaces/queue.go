package interfaces

type IQueue interface {
	Add(events ...IEvent)
	NextEvent() IEvent
	// DeleteOneOfTypeForNode deletes the first event of type for a node and returns it iff it was prior to the current node time.
	DeleteOneOfTypeForNode(eventType IEventType, node INode) IEvent
	Length() int
	CountEventTypesAndTimes() string
	SetWorld(world IWorld)
}
