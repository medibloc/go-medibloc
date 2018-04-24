package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestProtoReservedTask(t *testing.T) {
	taskType := "testType"
	payload := []byte("testPayload")
	timestamp := int64(1524192961)
	task := core.NewReservedTask(taskType, payload, timestamp)
	assert.NotNil(t, task)
	msg := task.ToProto()
	task2 := new(core.ReservedTask)
	assert.NoError(t, task2.FromProto(msg))
	assert.Equal(t, task, task2)
}

func TestProtoReservationQueue(t *testing.T) {
	data := []struct {
		taskType  string
		payload   []byte
		timestamp int64
	}{
		{"type1", []byte("payload1"), int64(1200000000)},
		{"type2", []byte("payload2"), int64(1300000000)},
		{"type3", []byte("payload3"), int64(1400000000)},
	}

	rq := core.NewEmptyReservationQueue(test.GenesisBlock.Storage())
	rq.BeginBatch()
	for i := range data {
		assert.NoError(t, rq.AddTask(core.NewReservedTask(
			data[i].taskType,
			data[i].payload,
			data[i].timestamp,
		)))
	}
	rq.Commit()

	rq2, err := core.LoadReservationQueue(rq.Storage(), rq.Hash())
	assert.NoError(t, err)
	assert.Equal(t, rq2.Tasks(), rq.Tasks())
	assert.Equal(t, rq2.Hash(), rq.Hash())
}

func TestPopTasksBefore(t *testing.T) {
	data := []struct {
		taskType  string
		payload   []byte
		timestamp int64
	}{
		{"type1", []byte("payload1"), int64(1200000000)},
		{"type2", []byte("payload2"), int64(1300000000)},
		{"type3", []byte("payload3"), int64(1400000000)},
	}

	rq := core.NewEmptyReservationQueue(test.GenesisBlock.Storage())
	rq.BeginBatch()
	for i := range data {
		assert.NoError(t, rq.AddTask(core.NewReservedTask(
			data[i].taskType,
			data[i].payload,
			data[i].timestamp,
		)))
	}
	rq.Commit()
	tasks := rq.PopTasksBefore(1350000000)
	assert.Equal(t, 2, len(tasks))
	assert.Equal(t, 1, len(rq.Tasks()))
	assert.Equal(t, data[0].taskType, tasks[0].TaskType())
	assert.Equal(t, data[0].payload, tasks[0].Payload())
	assert.Equal(t, data[0].timestamp, tasks[0].Timestamp())
	assert.Equal(t, data[1].taskType, tasks[1].TaskType())
	assert.Equal(t, data[1].payload, tasks[1].Payload())
	assert.Equal(t, data[1].timestamp, tasks[1].Timestamp())
	assert.Equal(t, data[2].taskType, rq.Tasks()[0].TaskType())
	assert.Equal(t, data[2].payload, rq.Tasks()[0].Payload())
	assert.Equal(t, data[2].timestamp, rq.Tasks()[0].Timestamp())
}
