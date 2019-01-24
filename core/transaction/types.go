package transaction

import (
	"errors"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// transaction types
const (
	TxOpTransfer            = "transfer"
	TxOpAddRecord           = "add_record"
	TxOpStake               = "stake"
	TxOpUnstake             = "unstake"
	TxOpAddCertification    = "add_certification"
	TxOpRevokeCertification = "revoke_certification"
	TxOpRegisterAlias       = "register_alias"
	TxOpDeregisterAlias     = "deregister_alias"
)

// Transaction's related to dpos
const (
	TxOpBecomeCandidate = "become_candidate"
	TxOpQuitCandidacy   = "quit_candidacy"
	TxOpVote            = "vote"
)

// DefaultTxMap is default map of transactions.
var DefaultTxMap = core.TxMapper{
	TxOpTransfer:            NewTransferTx,
	TxOpAddRecord:           NewAddRecordTx,
	TxOpStake:               NewStakeTx,
	TxOpUnstake:             NewUnstakeTx,
	TxOpAddCertification:    NewAddCertificationTx,
	TxOpRevokeCertification: NewRevokeCertificationTx,
	TxOpRegisterAlias:       NewRegisterAliasTx,
	TxOpDeregisterAlias:     NewDeregisterAliasTx,

	TxOpBecomeCandidate: NewBecomeCandidateTx,
	TxOpQuitCandidacy:   NewQuitCandidateTx,
	TxOpVote:            NewVoteTx,
}

// constants for staking and regeneration
const (
	MaxPayloadSize = 4096
)

// constants related to account alias
const (
	AliasLengthMinimum     = 12
	AliasLengthMaximum     = 16
	AliasCollateralMinimum = "1000000000000000000"
)

// constants related to dpos
const (
	CandidateCollateralMinimum = "1000000000000000000"
	VoteMaximum                = 15
)

// Errors related to execute transaction
var (
	ErrTooLargePayload          = errors.New("too large payload")
	ErrVoidTransaction          = errors.New("nothing to do with transaction")
	ErrInvalidAddress           = errors.New("invalid address")
	ErrBalanceNotEnough         = errors.New("balance is not enough")
	ErrCannotUseZeroValue       = errors.New("value should be larger than zero")
	ErrRecordHashInvalid        = errors.New("invalid record hash")
	ErrRecordAlreadyAdded       = errors.New("record hash already added")
	ErrNotFound                 = trie.ErrNotFound
	ErrCertHashInvalid          = errors.New("invalid certification hash")
	ErrCertReceivedAlreadyAdded = errors.New("hash of received cert already added")
	ErrCertIssuedAlreadyAdded   = errors.New("hash of issued cert already added")
	ErrCertAlreadyRevoked       = errors.New("cert to revoke has already been revoked")
	ErrCertAlreadyExpired       = errors.New("cert to revoke has already been expired")
	ErrCertRevokerInvalid       = errors.New("only issuer of the cert can revoke it")
	ErrAliasAlreadyTaken        = errors.New("already occupied alias")
	ErrAliasAlreadyHave         = errors.New("already have a alias name")
	ErrAliasCollateralLimit     = errors.New("not enough transaction value for alias collateral")
	ErrAliasFirstLetter         = errors.New("first letter of alias name should not be a number")
	ErrAliasInvalidChar         = errors.New("aliasname must contain only lowercase letters and numbers")
	ErrAliasLengthUnderMinimum  = errors.New("aliasname is too short")
	ErrAliasLengthExceedMaximum = errors.New("aliasname is too long ")
	ErrAliasNotExist            = errors.New("doesn't have any alias")
	ErrFailedToUnmarshalPayload = errors.New("cannot unmarshal tx payload")
	ErrFailedToMarshalPayload   = errors.New("cannot marshal tx payload to bytes")
	ErrCheckPayloadIntegrity    = errors.New("payload has invalid elements")
	ErrPointNotEnough           = errors.New("points are not enough")
)

// Errors related to dpos transactions (candidate, voting)
var (
	ErrOverMaxVote                  = errors.New("too many vote")
	ErrDuplicateVote                = errors.New("cannot vote multiple vote for same account")
	ErrAlreadyCandidate             = errors.New("account is already a candidate")
	ErrNotCandidate                 = errors.New("account is not a candidate")
	ErrNotEnoughCandidateCollateral = errors.New("candidate collateral is not enough")
)

// Payload is an interface of transaction payload.
type Payload interface {
	FromBytes(b []byte) error
	ToBytes() ([]byte, error)
}

// BytesToTransactionPayload convert byte slice to Payload
func BytesToTransactionPayload(bytes []byte, payload Payload) error {
	if err := payload.FromBytes(bytes); err != nil {
		return ErrFailedToUnmarshalPayload
	}
	b, err := payload.ToBytes()
	if err != nil {
		return ErrFailedToMarshalPayload
	}
	if !byteutils.Equal(bytes, b) {
		return ErrCheckPayloadIntegrity
	}
	return nil
}
