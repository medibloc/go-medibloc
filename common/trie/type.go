package trie

import (
	"errors"
)

// Trie errors string representation
var (
	ErrCannotPerformInBatch    = errors.New("cannot perform in batch")
	ErrNotBatching             = errors.New("not batching")
	ErrAlreadyPreparedTrie     = errors.New("cannot prepare again")
	ErrNotPrepared             = errors.New("not preparing")
	ErrCannotClonePreparedTrie = errors.New("cannot clone prepared trie")
)

// Serializable interface for serializing/deserializing
type Serializable interface {
	ToBytes() ([]byte, error)
	FromBytes([]byte) error
}

type Batcher interface {
	Prepare() error
	BeginBatch() error
	Commit() error
	RollBack() error
	Flush() error
	Reset() error
}

type BatchCallType func(batch Batcher) error

func PrepareCall(batch Batcher) error    { return batch.Prepare() }
func BeginBatchCall(batch Batcher) error { return batch.BeginBatch() }
func CommitCall(batch Batcher) error     { return batch.Commit() }
func RollbackCall(batch Batcher) error   { return batch.RollBack() }
func FlushCall(batch Batcher) error      { return batch.Flush() }
func ResetCall(batch Batcher) error      { return batch.Reset() }
