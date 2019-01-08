package event

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
	// TopicPendingTransaction Pending transactions in a pending pool.
	TopicPendingTransaction = "chain.pendingTransaction"
	// TopicFutureTransaction future transactions in a future pool
	TopicFutureTransaction = "chain.futureTransaction"
	// TopicRevertBlock revert block
	TopicRevertBlock = "chain.revertBlock"
	// TopicTransactionExecutionResult transaction execution result
	TopicTransactionExecutionResult = "chain.transactionResult"
	// TopicBlockAccepted new block accepted on the chain
	TopicAcceptedBlock = "chain.acceptedBlock"
	// TopicInvalidBLock block is invalid
	TopicInvalidBlock = "chain.invalidBlock"
)

// Type for account transaction result
const (
	TypeAccountTransactionExecution = "executedTransaction"
	TypeAccountTransactionPending   = "pendingTransaction"
	TypeAccountTransactionFuture    = "futureTransaction"
	TypeAccountTransactionDeleted   = "deletedTransaction"
)

// Event structure
type Event struct {
	Topic string
	Data  string
	Type  string
}

// Subscriber structure
type Subscriber struct {
	eventCh chan *Event
	topics  []string
}

func topicList() map[string]bool {
	var topicList = make(map[string]bool)
	topicList[TopicLibBlock] = true
	topicList[TopicNewTailBlock] = true
	topicList[TopicPendingTransaction] = true
	topicList[TopicFutureTransaction] = true
	topicList[TopicRevertBlock] = true
	topicList[TopicTransactionExecutionResult] = true
	topicList[TopicAcceptedBlock] = true
	topicList[TopicInvalidBlock] = true
	return topicList
}

// NewSubscriber creates new event subscriber
func NewSubscriber(size int, topics []string) (*Subscriber, error) {
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
	return &Subscriber{
		eventCh: eventCh,
		topics:  topics,
	}, nil
}

// EventChan returns event channel
func (s *Subscriber) EventChan() chan *Event {
	return s.eventCh
}

// Emitter structure
type Emitter struct {
	eventSubscribers *sync.Map
	eventCh          chan *Event
	quitCh           chan bool
	size             int
}

// NewEventEmitter creates new event emitter
func NewEventEmitter(size int) *Emitter {
	return &Emitter{
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
func (e *Emitter) Start() {
	logging.Console().Info("Starting Event Emitter")

	go e.loop()
}

// Stop event loop
func (e *Emitter) Stop() {
	logging.Console().Info("Stopping Event Emitter")

	e.quitCh <- true
}

// Trigger event
func (e *Emitter) Trigger(event *Event) {
	e.eventCh <- event
}

// Register event channel
func (e *Emitter) Register(subscribers ...*Subscriber) {
	for _, subscriber := range subscribers {
		for _, topic := range subscriber.topics {
			m, _ := e.eventSubscribers.LoadOrStore(topic, new(sync.Map))
			m.(*sync.Map).Store(subscriber, true)
		}
	}
}

// Deregister event channel
func (e *Emitter) Deregister(subscribers ...*Subscriber) {
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

func (e *Emitter) loop() {
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
				case key.(*Subscriber).eventCh <- event:
				default:
					logging.Console().Warn("timeout to dispatch event")
				}
				return true
			})
		}
	}
}
