package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// TransactionState is a structure for save transaction
type TransactionState struct {
	*trie.Batch
}

// NewTransactionState returns transaction state
func NewTransactionState(rootHash []byte, stor storage.Storage) (*TransactionState, error) {
	trieBatch, err := trie.NewBatch(rootHash, stor)
	if err != nil {
		return nil, err
	}
	return &TransactionState{
		Batch: trieBatch,
	}, nil
}

// Clone clones state
func (ts *TransactionState) Clone() (*TransactionState, error) {
	newBatch, err := ts.Batch.Clone()
	if err != nil {
		return nil, err
	}
	return &TransactionState{
		Batch: newBatch,
	}, nil
}

// GetTx returns transaction from transaction state
func (ts *TransactionState) GetTx(hash []byte) (*Transaction, error) {
	txBytes, err := ts.Batch.Get(hash)
	if err != nil {
		return nil, err
	}
	pbTx := new(corepb.Transaction)
	if err := proto.Unmarshal(txBytes, pbTx); err != nil {
		return nil, err
	}
	tx := new(Transaction)
	if err := tx.FromProto(pbTx); err != nil {
		return nil, err
	}
	return tx, nil
}

// Put put transaction to transaction state
func (ts *TransactionState) Put(tx *Transaction) error {
	pbTx, err := tx.ToProto()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to convert a transaction to proto.")
		return err
	}

	txBytes, err := proto.Marshal(pbTx)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"pb":  pbTx,
		}).Error("Failed to marshal proto.")
		return err
	}
	return ts.Batch.Put(tx.Hash(), txBytes)
}
