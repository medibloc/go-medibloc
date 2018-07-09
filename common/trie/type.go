package trie

import (
	"errors"
)

// Trie errors string representationn
var (
	ErrBeginAgainInBatch     = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching = errors.New("cannot clone on batching")
	ErrNotBatching           = errors.New("not batching")
)
