package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestProtoReservedTask(t *testing.T) {
	taskType := core.RtWithdrawType
	from := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	payload, err := core.NewRtWithdraw(util.NewUint128FromUint(0))
	assert.NoError(t, err)
	timestamp := int64(1524192961)
	task := core.NewReservedTask(taskType, from, payload, timestamp)
	assert.NotNil(t, task)
	msg, err := task.ToProto()
	assert.NoError(t, err)
	task2 := core.NewReservedTask("", common.Address{}, payload, int64(0))
	assert.NoError(t, task2.FromProto(msg))
	assert.Equal(t, task, task2)
}

func TestProtoReservationQueue(t *testing.T) {
	payloads := []*core.RtWithdraw{}
	for i := 0; i < 3; i++ {
		w, err := core.NewRtWithdraw(util.NewUint128FromUint(uint64(i)))
		assert.NoError(t, err)
		payloads = append(payloads, w)
	}
	data := []struct {
		taskType  string
		from      common.Address
		payload   core.Serializable
		timestamp int64
	}{
		{core.RtWithdrawType, common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c"), payloads[0], int64(1200000000)},
		{core.RtWithdrawType, common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"), payloads[1], int64(1300000000)},
		{core.RtWithdrawType, common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"), payloads[2], int64(1400000000)},
	}

	rq := core.NewEmptyReservationQueue(test.GenesisBlock.Storage())
	rq.BeginBatch()
	for i := range data {
		assert.NoError(t, rq.AddTask(core.NewReservedTask(
			data[i].taskType,
			data[i].from,
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
	payloads := []*core.RtWithdraw{}
	for i := 0; i < 3; i++ {
		w, err := core.NewRtWithdraw(util.NewUint128FromUint(uint64(i)))
		assert.NoError(t, err)
		payloads = append(payloads, w)
	}
	data := []struct {
		taskType  string
		from      common.Address
		payload   core.Serializable
		timestamp int64
	}{
		{core.RtWithdrawType, common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c"), payloads[0], int64(1200000000)},
		{core.RtWithdrawType, common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"), payloads[1], int64(1300000000)},
		{core.RtWithdrawType, common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"), payloads[2], int64(1400000000)},
	}

	rq := core.NewEmptyReservationQueue(test.GenesisBlock.Storage())
	rq.BeginBatch()
	for i := range data {
		assert.NoError(t, rq.AddTask(core.NewReservedTask(
			data[i].taskType,
			data[i].from,
			data[i].payload,
			data[i].timestamp,
		)))
	}
	rq.Commit()
	tasks := rq.PopTasksBefore(1350000000)
	assert.Equal(t, 2, len(tasks))
	assert.Equal(t, 1, len(rq.Tasks()))
	assert.Equal(t, data[0].taskType, tasks[0].TaskType())
	assert.Equal(t, data[0].from, tasks[0].From())
	assert.Equal(t, data[0].payload, tasks[0].Payload())
	assert.Equal(t, data[0].timestamp, tasks[0].Timestamp())

	assert.Equal(t, data[1].taskType, tasks[1].TaskType())
	assert.Equal(t, data[1].from, tasks[1].From())
	assert.Equal(t, data[1].payload, tasks[1].Payload())
	assert.Equal(t, data[1].timestamp, tasks[1].Timestamp())

	assert.Equal(t, data[2].taskType, rq.Tasks()[0].TaskType())
	assert.Equal(t, data[2].from, rq.Tasks()[0].From())
	assert.Equal(t, data[2].payload, rq.Tasks()[0].Payload())
	assert.Equal(t, data[2].timestamp, rq.Tasks()[0].Timestamp())
}
