package transaction

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// StakeTx is a structure for staking med
type StakeTx struct {
	*core.Transaction
	user   common.Address
	amount *util.Uint128
	size   int
}

var _ core.ExecutableTx = &StakeTx{}

// NewStakeTx returns NewTx
func NewStakeTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	if tx.Value().Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrCannotUseZeroValue
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}

	return &StakeTx{
		Transaction: tx,
		user:        tx.From(),
		amount:      tx.Value(),
		size:        size,
	}, nil
}

// Execute StakeTx
func (tx *StakeTx) Execute(b *core.Block) error {
	user, err := b.State().GetAccount(tx.user)
	if err != nil {
		return err
	}
	user.Balance, err = user.Balance.Sub(tx.amount)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}
	user.Staking, err = user.Staking.Add(tx.amount)
	if err != nil {
		return err
	}

	user.Points, err = user.Points.Add(tx.amount)
	if err != nil {
		return err
	}

	err = b.State().PutAccount(user)
	if err != nil {
		return err
	}

	voted := user.VotedSlice()

	// Add user's stake to candidates' votePower
	for _, v := range voted {
		err = b.State().DposState().AddVotePowerToCandidate(v, tx.amount)
		if err == trie.ErrNotFound {
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

// Bandwidth returns bandwidth.
func (tx *StakeTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

func (tx *StakeTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, tx.amount.DeepCopy()
}

func (tx *StakeTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}

// UnstakeTx is a structure for unstaking med
type UnstakeTx struct {
	*core.Transaction
	user   common.Address
	amount *util.Uint128
	size   int
}

var _ core.ExecutableTx = &UnstakeTx{}

// NewUnstakeTx returns UnstakeTx
func NewUnstakeTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}

	return &UnstakeTx{
		Transaction: tx,
		user:        tx.From(),
		amount:      tx.Value(),
		size:        size,
	}, nil
}

// Execute UnstakeTx
func (tx *UnstakeTx) Execute(b *core.Block) error {
	account, err := b.State().GetAccount(tx.user)
	if err != nil {
		return err
	}

	account.Staking, err = account.Staking.Sub(tx.amount)
	if err == util.ErrUint128Underflow {
		return core.ErrStakingNotEnough
	}
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to subtract staking.")
		return err
	}

	account.Unstaking, err = account.Unstaking.Add(tx.amount)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add unstaking.")
		return err
	}
	account.LastUnstakingTs = b.Timestamp()

	if account.Staking.Cmp(account.Points) < 0 {
		account.Points = account.Staking.DeepCopy()
	}

	voted := account.VotedSlice()

	err = b.State().PutAccount(account)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to put account.")
		return err
	}

	// Add user's staking to candidates' votePower
	for _, v := range voted {
		err = b.State().DposState().SubVotePowerToCandidate(v, tx.amount)
		if err == trie.ErrNotFound {
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

// Bandwidth returns bandwidth.
func (tx *UnstakeTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

func (tx *UnstakeTx) PointChange() (neg bool, abs *util.Uint128) {
	return true, tx.amount.DeepCopy()
}

func (tx *UnstakeTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}
