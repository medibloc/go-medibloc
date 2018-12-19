package core_test

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/medibloc/go-medibloc/util/byteutils"

	"github.com/medibloc/go-medibloc/util/testutil/blockutil"

	"github.com/medibloc/go-medibloc/util/testutil"

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
				Type:  "",
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
				Type:  "",
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

func TestTopicLibBlock(t *testing.T) {
	dynastySize := 6
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child().SignProposer()

	bm := seed.Med.BlockManager()
	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicLibBlock}
	subscriber := register(emitter, topics[0])

	newLIB := bb.Build()
	err := bm.PushBlockData(newLIB.BlockData)
	assert.NoError(t, err)

	tail := newLIB
	go func() {
		for i := 0; i < dynastySize*2/3; i++ {
			b := bb.Block(tail).Child().SignProposer().Build()
			err := bm.PushBlockData(b.BlockData)
			assert.NoError(t, err)
			tail = b
		}
		return
	}()

	event := <-subscriber.EventChan()
	assert.Equal(t, core.TopicLibBlock, event.Topic)
	assert.Equal(t, byteutils.Bytes2Hex(newLIB.Hash()), event.Data)
}

func TestTopicNewTailBlock(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child().SignProposer()

	bm := seed.Med.BlockManager()
	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicNewTailBlock}
	subscriber := register(emitter, topics[0])

	b := bb.Build()
	err := bm.PushBlockData(b.BlockData)
	assert.NoError(t, err)

	go func() {
		b = bb.Child().SignProposer().Build()
		err := bm.PushBlockData(b.BlockData)
		assert.NoError(t, err)
		return
	}()

	count := 1
	for i := 0; i < 2; i++ {
		count++
		event := <-subscriber.EventChan()
		block, err := bm.BlockByHeight(uint64(count))
		assert.NoError(t, err)
		assert.Equal(t, core.TopicNewTailBlock, event.Topic)
		assert.Equal(t, byteutils.Bytes2Hex(block.Hash()), event.Data)
	}
}

func TestTopicPendingTransaction(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child().SignProposer()

	tm := seed.Med.TransactionManager()
	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicPendingTransaction}
	subscriber := register(emitter, topics[0])

	tx := bb.Tx().RandomTx().Build()
	go func() {
		tm.Push(tx)
		return
	}()

	event := <-subscriber.EventChan()
	assert.Equal(t, core.TopicPendingTransaction, event.Topic)
	assert.Equal(t, byteutils.Bytes2Hex(tx.Hash()), event.Data)
}

func TestTopicRevertBlock(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child().SignProposer()

	bm := seed.Med.BlockManager()
	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicRevertBlock}
	subscriber := register(emitter, topics[0])

	b := bb.Build()
	forkedBlock := bb.Block(b).Child().SignProposer().Build()
	assert.NoError(t, bm.PushBlockData(b.BlockData))
	assert.NoError(t, bm.PushBlockData(forkedBlock.BlockData))
	assert.NoError(t, seed.WaitUntilBlockAcceptedOnChain(forkedBlock.Hash()))

	canonicalBlocks := make([]*core.Block, 2)
	b1 := bb.Block(b).Child().Tx().RandomTx().Execute().SignProposer().Build()
	b2 := bb.Block(b1).Child().SignProposer().Build()
	canonicalBlocks[0] = b1
	canonicalBlocks[1] = b2

	go func() {
		for _, v := range canonicalBlocks {
			err := bm.PushBlockData(v.BlockData)
			assert.NoError(t, err)
			assert.NoError(t, seed.WaitUntilBlockAcceptedOnChain(v.Hash()))
		}
		return
	}()

	event := <-subscriber.EventChan()
	assert.Equal(t, core.TopicRevertBlock, event.Topic)
	assert.Equal(t, byteutils.Bytes2Hex(forkedBlock.Hash()), event.Data)
}

func TestTopicTransactionExecutionResult(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child()

	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicTransactionExecutionResult}
	subscriber := register(emitter, topics[0])

	tx := bb.Tx().RandomTx().Build()

	go func() {
		b := bb.ExecuteTx(tx).SignProposer().Build()
		err := seed.Med.BlockManager().PushBlockData(b.BlockData)
		require.NoError(t, err)
		return
	}()

	event := <-subscriber.EventChan()
	assert.Equal(t, core.TopicTransactionExecutionResult, event.Topic)
	assert.Equal(t, byteutils.Bytes2Hex(tx.Hash()), event.Data)
}

func TestTopicAcceptedBlock(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child()
	b := bb.SignProposer().Build()

	emitter := seed.Med.EventEmitter()
	topics := []string{core.TopicAcceptedBlock}
	subscriber := register(emitter, topics[0])

	err := seed.Med.BlockManager().PushBlockData(b.BlockData)
	require.NoError(t, err)

	event := <-subscriber.EventChan()
	assert.Equal(t, core.TopicAcceptedBlock, event.Topic)
	assert.Equal(t, byteutils.Bytes2Hex(b.Hash()), event.Data)
}

func TestTypeAccountTransaction(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.
		TokenDist).Block(seed.GenesisBlock()).Child()

	tx := bb.Tx().RandomTx().Build()

	emitter := seed.Med.EventEmitter()
	topics := []string{tx.From().Hex(), tx.To().Hex()}
	subscriber := register(emitter, topics[0], topics[1])

	b := bb.ExecuteTx(tx).SignProposer().Build()
	go func() {
		seed.Med.TransactionManager().Push(tx)
		err := bm.PushBlockData(b.BlockData)
		assert.NoError(t, err)
		err = seed.WaitUntilTailHeight(b.Height())
		require.NoError(t, err)
		return
	}()

	fromEventCount := 2
	toEventCount := 2
	for i := 0; i < 4; i++ {
		event := <-subscriber.EventChan()
		assert.Equal(t, byteutils.Bytes2Hex(tx.Hash()), event.Data)
		if event.Topic == topics[0] {
			fromEventCount = fromEventCount - 1
		} else {
			toEventCount = toEventCount - 1
		}
	}

	assert.Equal(t, 0, fromEventCount, toEventCount)
}
