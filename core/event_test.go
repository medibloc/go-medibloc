package core_test

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/stretchr/testify/assert"
)

func register(emitter *core.EventEmitter, topics ...string) *core.EventSubscriber {
	eventSubscriber, _ := core.NewEventSubscriber(1024, topics)
	emitter.Register(eventSubscriber)
	return eventSubscriber
}

func TestEventEmitter(t *testing.T) {
	emitter := core.NewEventEmitter(1024)
	emitter.Start()

	topics := []string{core.TopicPendingTransaction, core.TopicLibBlock}

	subscriber1 := register(emitter, topics[0], topics[1])

	totalEventCount := 1000
	eventCountDist := make(map[string]int)

	// Run only one go routine
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		defer wg.Done()

		rand.Seed(time.Now().UnixNano())

		for i := 0; i < totalEventCount; i++ {
			topic := topics[rand.Intn(len(topics))]
			eventCountDist[topic] = eventCountDist[topic] + 1

			e := &core.Event{
				Topic: topic,
				Data:  fmt.Sprintf("%d", i),
			}
			emitter.Trigger(e)
		}
	}()

	// Check buffered channel
	eventCh1 := subscriber1.EventChan()
	eventCount1, eventCount2 := 0, 0
	receiving := true

	for receiving {
		e := <-eventCh1
		if e.Topic == topics[0] {
			eventCount1++
		} else if e.Topic == topics[1] {
			eventCount2++
		}
		if e.Data == strconv.Itoa(totalEventCount-1) {
			receiving = false
		}
	}

	expectedEventCount1 := eventCountDist[topics[0]]
	expectedEventCount2 := eventCountDist[topics[1]]
	assert.Equal(t, eventCount1, expectedEventCount1)
	assert.Equal(t, eventCount2, expectedEventCount2)

	wg.Wait()
	emitter.Stop()
}

func TestEventEmitterWithRunningRegDereg(t *testing.T) {
	emitter := core.NewEventEmitter(1024)
	emitter.Start()

	topics := []string{core.TopicPendingTransaction}

	subscriber1 := register(emitter, topics[0])
	subscriber2 := register(emitter, topics[0])

	totalEventCount := 1000
	eventCountDist := make(map[string]int)

	// Run two go routines
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()

		rand.Seed(time.Now().UnixNano())

		for i := 0; i < totalEventCount; i++ {
			topic := topics[rand.Intn(len(topics))]
			eventCountDist[topic] = eventCountDist[topic] + 1

			e := &core.Event{
				Topic: topic,
				Data:  fmt.Sprintf("%d", i),
			}
			emitter.Trigger(e)
		}
	}()

	// check buffered channel
	eventCh1, eventCh2 := subscriber1.EventChan(), subscriber2.EventChan()
	eventCount1, eventCount2 := 0, 0

	emitter.Deregister(subscriber2)
	time.Sleep(100)
	go func() {
		defer wg.Done()
		quit := time.NewTimer(time.Second * 1)
		defer quit.Stop()

		for {
			select {
			case <-quit.C:
				return
			case <-eventCh1:
				eventCount1++
			case <-eventCh2:
				eventCount2++
			}
		}
	}()

	wg.Wait()

	assert.Equal(t, eventCount1, eventCountDist[topics[0]])
	assert.Equal(t, eventCountDist[topics[0]] > eventCount2, true)

	emitter.Stop()
}
