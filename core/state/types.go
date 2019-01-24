package corestate

import (
	"errors"
	"time"

	"github.com/medibloc/go-medibloc/common/trie"
)

// constants
const (
	UnstakingWaitDuration    = 7 * 24 * time.Hour
	PointsRegenerateDuration = 7 * 24 * time.Hour
)

// errors
var (
	ErrElapsedTimestamp = errors.New("cannot calculate points for elapsed timestamp")
	ErrNotFound         = trie.ErrNotFound
	ErrUnauthorized     = errors.New("unauthorized request")
)
