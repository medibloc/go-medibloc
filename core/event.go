package core

import (
	"sync"

	"github.com/medibloc/go-medibloc/common"

	"github.com/medibloc/go-medibloc/util/logging"
)

// TODO @ggomma check whether event emitter exists
const (
	// TopicLibBlock latest irreversible block.
	TopicLibBlock = "chain.latestIrreversibleBlock"

	// TopicNewTailBlock new tail block set
	TopicNewTailBlock = "chain.newTailBlock"

	// TopicPendingTransaction Pending transactions in a transaction pool.
	TopicPendingTransaction = "chain.pendingTransaction"

	// TopicRevertBlock revert block
	TopicRevertBlock = "chain.revertBlock"

	// TopicTransactionExecutionResult transaction execution result
	TopicTransactionExecutionResult = "chain.transactionResult"
)

// Type for account transaction result
const (
	TypeAccountTransactionExecution = "executedTransaction"
	TypeAccountTransactionPending   = "pendingTransaction"
	TypeAccountTransactionDeleted   = "deletedTransaction"
)

// Event structure
type Event struct {
	Topic string
	Data  string
	Type  string
}

// EventSubscriber structure
type EventSubscriber struct {
	eventCh chan *Event
	topics  []string
}

func topicList() map[string]bool {
	var topicList = make(map[string]bool)
	topicList[TopicLibBlock] = true
	topicList[TopicNewTailBlock] = true
	topicList[TopicPendingTransaction] = true
	topicList[TopicRevertBlock] = true
	topicList[TopicTransactionExecutionResult] = true
	return topicList
}

// NewEventSubscriber creates new event subscriber
func NewEventSubscriber(size int, topics []string) (*EventSubscriber, error) {
	topicList := topicList()

	for _, topic := range topics {
		if common.IsHexAddress(topic) {
			continue
		}

		if topicList[topic] != true {
			return nil, ErrWrongEventTopic
		}
	}

	eventCh := make(chan *Event, size)
	return &EventSubscriber{
		eventCh: eventCh,
		topics:  topics,
	}, nil
}

// EventChan returns event channel
func (s *EventSubscriber) EventChan() chan *Event {
	return s.eventCh
}

// EventEmitter structure
type EventEmitter struct {
	eventSubscribers *sync.Map
	eventCh          chan *Event
	quitCh           chan bool
	size             int
}

// NewEventEmitter creates new event emitter
func NewEventEmitter(size int) *EventEmitter {
	return &EventEmitter{
		eventSubscribers: new(sync.Map),
		eventCh:          make(chan *Event, size),
		quitCh:           make(chan bool),
		size:             size,
	}
}

/*
	NewEventEmitter
		- eventSubscribers
			- topic #1
				- subscriber #1 : true
				- subscriber #2 : true
			- topic #2
				- subscriber #2 : true
*/

// Start event loop
func (e *EventEmitter) Start() {
	logging.Console().Info("Starting Event Emitter")

	go e.loop()
}

// Stop event loop
func (e *EventEmitter) Stop() {
	logging.Console().Info("Stopping Event Emitter")

	e.quitCh <- true
}

// Trigger event
func (e *EventEmitter) Trigger(event *Event) {
	e.eventCh <- event
}

// Register event channel
func (e *EventEmitter) Register(subscribers ...*EventSubscriber) {
	for _, subscriber := range subscribers {
		for _, topic := range subscriber.topics {
			m, _ := e.eventSubscribers.LoadOrStore(topic, new(sync.Map))
			m.(*sync.Map).Store(subscriber, true)
		}
	}
}

// Deregister event channel
func (e *EventEmitter) Deregister(subscribers ...*EventSubscriber) {
	for _, subscriber := range subscribers {
		for _, topic := range subscriber.topics {
			m, _ := e.eventSubscribers.Load(topic)
			// If the topic doesn't hold any subscriber
			if m == nil {
				continue
			}
			m.(*sync.Map).Delete(subscriber)
		}
	}
}

func (e *EventEmitter) loop() {
	logging.Console().Info("Started Event Emitter")

	// timerChan := time.NewTicker(time.Second).C
	// defer timerChan.Stop()
	for {
		select {
		// case <- timerChan:
		// TODO : Metrics
		case <-e.quitCh:
			logging.Console().Info("Stopped Event Emitter")
			return
		case event := <-e.eventCh:
			topic := event.Topic
			subscribers, ok := e.eventSubscribers.Load(topic)
			// If topic subscriber doesn't exist, continue
			if !ok {
				continue
			}

			subs, _ := subscribers.(*sync.Map)
			subs.Range(func(key, value interface{}) bool {
				select {
				case key.(*EventSubscriber).eventCh <- event:
				default:
					logging.Console().Warn("timeout to dispatch event")
				}
				return true
			})
		}
	}
}
