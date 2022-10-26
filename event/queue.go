package event

import (
	"ethattacksim/interfaces"
	"fmt"
	"math"
)

type Queue struct {
	events                    []interfaces.IEvent
	newEvents                 []interfaces.IEvent
	earliestNewEventTimestamp int64
	world                     interfaces.IWorld
}

func NewQueue() *Queue {
	return &Queue{make([]interfaces.IEvent, 0, 1000), make([]interfaces.IEvent, 0, 200), math.MaxInt64, nil}
}

func (q *Queue) Add(events ...interfaces.IEvent) {
	for _, event := range events {
		if event.Time() <= q.world.EndTime() {
			q.newEvents = append(q.newEvents, event)
			if q.earliestNewEventTimestamp > event.Time() {
				q.earliestNewEventTimestamp = event.Time()
			}
		}
	}
}

// if time is equal, the first event found is returned
func (q *Queue) NextEvent() interfaces.IEvent {
	q.fitNewEventsToQueue()
	nextEv := q.events[0]
	q.events = q.events[1:]
	return nextEv
}

func (q *Queue) fitNewEventsToQueue() {
	// if events queue is empty but new events queue is not
	if len(q.events) == 0 && len(q.newEvents) > 0 {
		// first sort new events
		q.newEvents = MergeSort(q.newEvents)
		// simply make new events the new events queue
		q.events = q.newEvents
		q.newEvents = make([]interfaces.IEvent, 0, 200)
		q.earliestNewEventTimestamp = math.MaxInt64
		return
	}

	// if no new events or earliest new event timestamp after earliest event timestamp, simply return, else merge lists
	if len(q.newEvents) > 0 && len(q.events) > 0 && q.earliestNewEventTimestamp <= q.events[0].Time() {
		// first sort new events
		q.newEvents = MergeSort(q.newEvents)

		// last elem from queue is earlier than first from new events, simply append both lists
		if q.events[len(q.events)-1].Time() < q.newEvents[0].Time() {
			q.events = append(q.events, q.newEvents...)
			q.newEvents = make([]interfaces.IEvent, 0, 200)
			q.earliestNewEventTimestamp = math.MaxInt64
			return
		}

		// merge queue with sorted new events
		q.events = Merge(q.events, q.newEvents)
		q.newEvents = make([]interfaces.IEvent, 0, 200)
		q.earliestNewEventTimestamp = math.MaxInt64
	}
	return
}

func MergeSort(src []interfaces.IEvent) []interfaces.IEvent {
	if len(src) <= 1 {
		return src
	}
	mid := len(src) / 2
	return Merge(MergeSort(src[:mid]), MergeSort(src[mid:]))
}

func Merge(left, right []interfaces.IEvent) []interfaces.IEvent {
	result := make([]interfaces.IEvent, 0, len(left)+len(right))
	var l, r int
	for l < len(left) || r < len(right) {
		if l < len(left) && r < len(right) {
			if left[l].Time() <= right[r].Time() {
				result = append(result, left[l])
				l++
			} else {
				result = append(result, right[r])
				r++
			}
		} else if l < len(left) {
			result = append(result, left[l:]...)
			break
		} else if r < len(right) {
			result = append(result, right[r:]...)
			break
		}
	}
	return result
}

func (q *Queue) DeleteOneOfTypeForNode(eventType interfaces.IEventType, node interfaces.INode) interfaces.IEvent {
	if len(q.events) > len(q.newEvents) {
		for i := 0; i < len(q.events); i++ {
			qEvent := q.events[i]
			if qEvent.Type() == eventType && qEvent.TargetId() == node.Id() {
				q.events = append(q.events[:i], q.events[i+1:]...)
				if qEvent.Time() < node.Time() {
					return qEvent
				}
				return nil
			}
			if i < len(q.newEvents) {
				nEvent := q.newEvents[i]
				if nEvent.Type() == eventType && nEvent.TargetId() == node.Id() {
					q.newEvents = append(q.newEvents[:i], q.newEvents[i+1:]...)
					if nEvent.Time() < node.Time() {
						return nEvent
					}
					return nil
				}
			}
		}
	} else {
		for i := 0; i < len(q.newEvents); i++ {
			nEvent := q.newEvents[i]
			if nEvent.Type() == eventType && nEvent.TargetId() == node.Id() {
				q.newEvents = append(q.newEvents[:i], q.newEvents[i+1:]...)
				if nEvent.Time() < node.Time() {
					return nEvent
				}
				return nil
			}
			if i < len(q.events) {
				qEvent := q.events[i]
				if qEvent.Type() == eventType && qEvent.TargetId() == node.Id() {
					q.events = append(q.events[:i], q.events[i+1:]...)
					if qEvent.Time() < node.Time() {
						return qEvent
					}
					return nil
				}
			}
		}
	}
	return nil
}

func (q *Queue) Length() int {
	return len(q.events) + len(q.newEvents)
}

func (q *Queue) CountEventTypesAndTimes() string {
	// for debugging purposes
	counts := make(map[string]int)
	counts["pastEvents"] = 0
	for i := 0; i < len(q.events); i++ {
		evType := fmt.Sprintf("%v", q.events[i].Type())
		if _, exists := counts[evType]; !exists {
			counts[evType] = 1
		} else {
			counts[evType]++
		}
		if q.events[i].Time() < q.world.Time() {
			counts["pastEvents"]++
		}
	}
	for i := 0; i < len(q.newEvents); i++ {
		evType := fmt.Sprintf("%v", q.events[i].Type())
		if _, exists := counts[evType]; !exists {
			counts[evType] = 1
		} else {
			counts[evType]++
		}
		if q.events[i].Time() < q.world.Time() {
			counts["pastEvents"]++
		}
	}
	ret := ""
	for countName, count := range counts {
		ret += fmt.Sprintf("%v: %v\n", countName, count)
	}
	return ret
}

func (q *Queue) SetWorld(world interfaces.IWorld) {
	q.world = world
	q.events = make([]interfaces.IEvent, 0, len(world.Nodes())*10000)
}
