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
