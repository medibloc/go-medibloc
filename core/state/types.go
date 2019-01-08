package coreState

import (
	"errors"
	"time"

	"github.com/medibloc/go-medibloc/common/trie"
)

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

// constants
const (
	UnstakingWaitDuration    = 7 * 24 * time.Hour
	PointsRegenerateDuration = 7 * 24 * time.Hour
)

// errors
var (
	ErrElapsedTimestamp             = errors.New("cannot calculate points for elapsed timestamp")
	ErrNotFound                     = trie.ErrNotFound
	ErrStakingNotEnough             = errors.New("staking is not enough")
	ErrPointNotEnough               = errors.New("points are not enough")
	ErrUnauthorized                 = errors.New("unauthorized request")
	ErrCannotConvertReceipt         = errors.New("proto message cannot be converted into Receipt")
	ErrInvalidReceiptToProto        = errors.New("receipt cannot be converted into proto")
	ErrCannotConvertTransaction     = errors.New("proto message cannot be converted into Transaction")
	ErrTransactionSignatureNotExist = errors.New("signature does not exist in the tx")
	ErrPayerSignatureNotExist       = errors.New("payer signature does not exist in the tx")
	ErrInvalidTransactionHash       = errors.New("invalid transaction hash")
	ErrCannotRecoverPayer           = errors.New("failed to recover payer from payer sign")
	ErrInvalidTxChainID             = errors.New("invalid transaction chainID")
)

type State interface {
	Clone() (State, error)
	Prepare() error
	BeginBatch() error
	Commit() error
	RollBack() error
	Reset() error
	Flush() error
	//RootBytes() ([]byte, error)
}
