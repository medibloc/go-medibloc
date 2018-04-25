package core

import (
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/crypto/sha3"
)

// ReservedTasks represents list of ReservedTask objects
type ReservedTasks []*ReservedTask

// Less for sort.Interface
func (tasks ReservedTasks) Less(i, j int) bool { return tasks[i].timestamp < tasks[j].timestamp }

// Len for sort.Interface
func (tasks ReservedTasks) Len() int {
	return len(tasks)
}

// Swap for sort.Interface
func (tasks ReservedTasks) Swap(i, j int) {
	tasks[i], tasks[j] = tasks[j], tasks[i]
}

// ReservationQueue manages multiple instances with ReservedTask type
type ReservationQueue struct {
	tasks   ReservedTasks
	hash    common.Hash
	storage storage.Storage

	batching bool
	snapshot common.Hash
}

// NewEmptyReservationQueue returns empty reserved queue
func NewEmptyReservationQueue(storage storage.Storage) *ReservationQueue {
	return &ReservationQueue{
		tasks:   ReservedTasks{},
		hash:    common.ZeroHash(),
		storage: storage,
	}
}

// ToProto converts ReservationQueue.task to corepb.ReservedTasks
func (rq *ReservationQueue) ToProto() (proto.Message, error) {
	pbTasks := new(corepb.ReservedTasks)
	for _, t := range rq.tasks {
		pbTask, err := t.ToProto()
		if err != nil {
			return nil, err
		}
		pbTasks.Tasks = append(pbTasks.Tasks, pbTask.(*corepb.ReservedTask))
	}
	return pbTasks, nil
}

// FromProto converts corepb.ReservedTasks to ReservationQueue.task
func (rq *ReservationQueue) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.ReservedTasks); ok {
		for _, pt := range msg.Tasks {
			t := new(ReservedTask)
			if err := t.FromProto(pt); err != nil {
				return err
			}
			rq.tasks = append(rq.tasks, t)
		}
		return nil
	}
	return ErrCannotConvertResevedTasks
}

// LoadReservationQueue loads reservation queue by hash from storage
func LoadReservationQueue(storage storage.Storage, hash common.Hash) (*ReservationQueue, error) {
	if common.IsZeroHash(hash) {
		return NewEmptyReservationQueue(storage), nil
	}
	b, err := storage.Get(hash.Bytes())
	if err != nil {
		return nil, err
	}
	rq := NewEmptyReservationQueue(storage)
	pbTasks := new(corepb.ReservedTasks)
	if err := proto.Unmarshal(b, pbTasks); err != nil {
		return nil, err
	}
	if err := rq.FromProto(pbTasks); err != nil {
		return nil, err
	}
	hashCalc, err := rq.calcHash()
	if err != nil {
		return nil, err
	}
	if !byteutils.Equal(hash.Bytes(), hashCalc) {
		return nil, ErrInvalidReservationQueueHash
	}
	rq.hash = hash
	return rq, nil
}

// Tasks returns rq.tasks
func (rq *ReservationQueue) Tasks() ReservedTasks {
	return rq.tasks
}

// Storage returns rq.storage
func (rq *ReservationQueue) Storage() storage.Storage {
	return rq.storage
}

// Hash returns rq.hash
func (rq *ReservationQueue) Hash() common.Hash {
	return rq.hash
}

// BeginBatch sets batching true to add task items
func (rq *ReservationQueue) BeginBatch() error {
	if rq.batching {
		return ErrReservationQueueAlreadyBatching
	}
	rq.batching = true
	return nil
}

// Commit saves new hash value and tasks list to storage
func (rq *ReservationQueue) Commit() error {
	if err := rq.save(); err != nil {
		return err
	}
	rq.batching = false
	rq.snapshot = common.ZeroHash()
	return nil
}

// RollBack reverts hash and reload tasks list
func (rq *ReservationQueue) RollBack() error {
	reloadedRq, err := LoadReservationQueue(rq.storage, rq.snapshot)
	if err != nil {
		return err
	}
	rq.tasks = reloadedRq.tasks
	rq.batching = false
	rq.snapshot = common.ZeroHash()
	return nil
}

// AddTask adds a task t in rq, sorts and calculate new hash
func (rq *ReservationQueue) AddTask(t *ReservedTask) error {
	if !rq.batching {
		return ErrReservationQueueNotBatching
	}
	rq.tasks = append(rq.tasks, t)
	sort.Sort(rq.tasks)
	hash, err := rq.calcHash()
	if err != nil {
		return err
	}
	rq.hash = common.BytesToHash(hash)
	return nil
}

// PopTasksBefore pop tasks of which timestamp is
func (rq *ReservationQueue) PopTasksBefore(timestamp int64) []*ReservedTask {
	tasks := []*ReservedTask{}
	t := rq.popOnlyBefore(timestamp)
	for t != nil {
		tasks = append(tasks, t)
		t = rq.popOnlyBefore(timestamp)
	}
	return tasks
}

// Peek returns first index task without removing it from listing
func (rq *ReservationQueue) Peek() *ReservedTask {
	if len(rq.tasks) == 0 {
		return nil
	}
	return rq.tasks[0]
}

func (rq *ReservationQueue) pop() *ReservedTask {
	t := rq.tasks[0]
	rq.tasks = rq.tasks[1:]
	return t
}

func (rq *ReservationQueue) popOnlyBefore(timestamp int64) *ReservedTask {
	if len(rq.tasks) == 0 {
		return nil
	}

	head := rq.Peek()
	if head.timestamp <= timestamp {
		return rq.pop()
	}
	return nil
}

func (rq *ReservationQueue) save() error {
	msg, err := rq.ToProto()
	if err != nil {
		return err
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return rq.storage.Put(rq.hash.Bytes(), b)
}

func (rq *ReservationQueue) calcHash() ([]byte, error) {
	hasher := sha3.New256()
	for _, t := range rq.tasks {
		taskBytes, err := t.calcHash()
		if err != nil {
			return nil, err
		}
		hasher.Write(taskBytes)
	}
	return hasher.Sum(nil), nil
}
