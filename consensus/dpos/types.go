package dpos

import (
	"errors"
	"time"
)

// Consensus properties.
const (
	BlockInterval              = 15 * time.Second
	DynastyInterval            = 210 * BlockInterval
	DynastySize                = 21
	IterationInDynastyInterval = DynastyInterval / DynastySize
	ConsensusSize              = DynastySize*2/3 + 1
	MinMintDuration            = 2 * time.Second

	miningTickInterval = time.Second
)

// Error types of dpos package.
var (
	ErrInvalidBlockInterval         = errors.New("invalid block interval")
	ErrInvalidBlockProposer         = errors.New("invalid block proposer")
	ErrInvalidBlockForgeTime        = errors.New("invalid time to forge block")
	ErrFoundNilProposer             = errors.New("found a nil proposer")
	ErrInvalidProtoToConsensusState = errors.New("protobuf message cannot be converted into ConsensusState")
	ErrBlockMintedInNextSlot        = errors.New("cannot mint block now, there is a block minted in current slot")
	ErrWaitingBlockInLastSlot       = errors.New("cannot mint block now, waiting for last block")
	ErrInvalidDynastySize           = errors.New("invalid dynasty size")
)
