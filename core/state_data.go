// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
)

//DataState is a structure for save data
type DataState struct {
	TxsState           *trie.Batch
	RecordsState       *trie.Batch
	CertificationState *trie.Batch

	storage storage.Storage
}

//NewDataState returns data state
func NewDataState(rootBytes []byte, stor storage.Storage) (*DataState, error) {
	pbDataState := new(corepb.DataState)
	if err := proto.Unmarshal(rootBytes, pbDataState); err != nil {
		return nil, err
	}

	ts, err := trie.NewBatch(pbDataState.TxStateRootHash, stor)
	if err != nil {
		return nil, err
	}
	rs, err := trie.NewBatch(pbDataState.RecordStateRootHash, stor)
	if err != nil {
		return nil, err
	}
	cs, err := trie.NewBatch(pbDataState.CertificationStateRootHash, stor)
	if err != nil {
		return nil, err
	}
	return &DataState{
		TxsState:           ts,
		RecordsState:       rs,
		CertificationState: cs,
		storage:            stor,
	}, nil
}

//Commit commit data state
func (ds *DataState) Commit() error {
	if err := ds.TxsState.Commit(); err != nil {
		return err
	}
	if err := ds.RecordsState.Commit(); err != nil {
		return err
	}
	if err := ds.CertificationState.Commit(); err != nil {
		return err
	}
	return nil
}

//RollBack rollbacks batch
func (ds *DataState) RollBack() error {
	if err := ds.TxsState.RollBack(); err != nil {
		return err
	}
	if err := ds.RecordsState.RollBack(); err != nil {
		return err
	}
	if err := ds.CertificationState.RollBack(); err != nil {
		return err
	}
	return nil
}

//BeginBatch start batching
func (ds *DataState) BeginBatch() error {
	if err := ds.TxsState.BeginBatch(); err != nil {
		return err
	}
	if err := ds.RecordsState.BeginBatch(); err != nil {
		return err
	}
	if err := ds.CertificationState.BeginBatch(); err != nil {
		return err
	}
	return nil
}

//RootBytes returns root bytes
func (ds *DataState) RootBytes() ([]byte, error) {
	pbDataState := &corepb.DataState{
		TxStateRootHash:            ds.TxsState.RootHash(),
		RecordStateRootHash:        ds.RecordsState.RootHash(),
		CertificationStateRootHash: ds.CertificationState.RootHash(),
	}
	return proto.Marshal(pbDataState)
}

//Clone copy data state
func (ds *DataState) Clone() (*DataState, error) {
	rb, err := ds.RootBytes()
	if err != nil {
		return nil, err
	}
	return NewDataState(rb, ds.storage)
}

//GetRecord returns record data
func (ds *DataState) GetRecord(hash []byte) (*corepb.Record, error) {
	recordBytes, err := ds.RecordsState.Get(hash)
	if err != nil {
		return nil, err
	}
	pbRecord := new(corepb.Record)
	if err := proto.Unmarshal(recordBytes, pbRecord); err != nil {
		return nil, err
	}
	return pbRecord, nil
}

//GetTx returns transaction data
func (ds *DataState) GetTx(txHash []byte) ([]byte, error) {
	return ds.TxsState.Get(txHash)
}

//PutTx put transaction to data state
func (ds *DataState) PutTx(txHash []byte, txBytes []byte) error {
	return ds.TxsState.Put(txHash, txBytes)
}

//Certification returns certification for hash
func (ds *DataState) Certification(hash []byte) (*corepb.Certification, error) {
	certBytes, err := ds.CertificationState.Get(hash)
	if err != nil {
		return nil, err
	}
	pbCert := new(corepb.Certification)
	err = proto.Unmarshal(certBytes, pbCert)
	if err != nil {
		return nil, err
	}
	return pbCert, nil
}
