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

// Batcher is an interface for batch commands.
type Batcher interface {
	Prepare() error
	BeginBatch() error
	Commit() error
	RollBack() error
	Flush() error
	Reset() error
}

// BatchCallType is a type of batch call function.
type BatchCallType func(batch Batcher) error

// PrepareCall is a prepare call.
func PrepareCall(batch Batcher) error { return batch.Prepare() }

// BeginBatchCall is a begin batch call.
func BeginBatchCall(batch Batcher) error { return batch.BeginBatch() }

// CommitCall is a commit call.
func CommitCall(batch Batcher) error { return batch.Commit() }

// RollbackCall is a rollback call.
func RollbackCall(batch Batcher) error { return batch.RollBack() }

// FlushCall is a flush call.
func FlushCall(batch Batcher) error { return batch.Flush() }

// ResetCall is a reset call.
func ResetCall(batch Batcher) error { return batch.Reset() }
