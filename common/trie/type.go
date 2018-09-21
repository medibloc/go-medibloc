package trie

import (
	"errors"
)

// Trie errors string representationn
var (
	ErrBeginAgainInBatch       = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching   = errors.New("cannot clone on batching")
	ErrNotBatching             = errors.New("not batching")
	ErrAlreadyPreparedTrie     = errors.New("cannot prepare again")
	ErrNotPrepared             = errors.New("not preparing")
	ErrCannotClonePreparedTrie = errors.New("cannot clone prepared trie")
)
