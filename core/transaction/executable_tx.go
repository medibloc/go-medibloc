package transaction

import (
	"github.com/medibloc/go-medibloc/common"
	dState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
)

// DefaultTxMap is default map of transactions.
var DefaultTxMap = TxFactory{
	coreState.TxOpTransfer:            NewTransferTx,
	coreState.TxOpAddRecord:           NewAddRecordTx,
	coreState.TxOpStake:               NewStakeTx,
	coreState.TxOpUnstake:             NewUnstakeTx,
	coreState.TxOpAddCertification:    NewAddCertificationTx,
	coreState.TxOpRevokeCertification: NewRevokeCertificationTx,
	coreState.TxOpRegisterAlias:       NewRegisterAliasTx,
	coreState.TxOpDeregisterAlias:     NewDeregisterAliasTx,

	dState.TxOpBecomeCandidate: NewBecomeCandidateTx,
	dState.TxOpQuitCandidacy:   NewQuitCandidateTx,
	dState.TxOpVote:            NewVoteTx,
}

//ExecutableTx is a structure to execute transaction
type ExecutableTx struct {
	*coreState.Transaction
	Executable
}

//NewExecutableTx returns new executable transaction
func NewExecutableTx(transaction *coreState.Transaction) (*ExecutableTx, error) {
	newTxFunc, ok := DefaultTxMap[transaction.TxType()]
	if !ok {
		return nil, ErrTxTypeInvalid
	}

	exeTx, err := newTxFunc(transaction)
	if err != nil {
		return nil, err
	}

	return &ExecutableTx{
		Transaction: transaction,
		Executable:  exeTx,
	}, nil
}

//CalcPoints returns required points to execute transaction
func (exeTx *ExecutableTx) CalcPoints(price common.Price) (*util.Uint128, error) {
	return exeTx.Bandwidth().CalcPoints(price)
}

// CheckAccountPoints compare given transaction's required bandwidth with the account's remaining bandwidth
func (exeTx *ExecutableTx) CheckAccountPoints(acc *coreState.Account, price common.Price, offset *common.Bandwidth) error {
	var err error
	avail := acc.Points
	switch exeTx.TxType() {
	case coreState.TxOpStake:
		avail, err = acc.Points.Add(exeTx.Value())
		if err != nil {
			return err
		}
	case coreState.TxOpUnstake:
		avail, err = acc.Points.Sub(exeTx.Value())
		if err == util.ErrUint128Underflow {
			return coreState.ErrStakingNotEnough
		}
		if err != nil {
			return err
		}
	}

	bw := exeTx.Bandwidth()
	if offset != nil {
		bw.Add(offset)
	}
	points, err := bw.CalcPoints(price)
	if err != nil {
		return err
	}

	if avail.Cmp(points) < 0 {
		return coreState.ErrPointNotEnough
	}
	return nil
}
