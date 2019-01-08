package coreState

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Prefixes for Account's Data trie
const (
	RecordsPrefix      = "r_"    // records
	CertReceivedPrefix = "cr_"   // certs received
	CertIssuedPrefix   = "ci_"   // certs issued
	AliasKey           = "alias" // alias key for data trie
)

// Account default item in state
type Account struct {
	Address     common.Address // address account key
	Balance     *util.Uint128  // balance account's coin amount
	Nonce       uint64         // nonce account sequential number
	Staking     *util.Uint128  // account's staking amount
	Voted       *trie.Batch    // voted candidates key: candidateID, value: candidateID
	CandidateID []byte         //

	Points       *util.Uint128
	LastPointsTs int64

	Unstaking       *util.Uint128
	LastUnstakingTs int64

	Data *trie.Batch // contain records, certReceived, certIssued

	Storage storage.Storage
}

func newAccount(stor storage.Storage) (*Account, error) {
	voted, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}
	data, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	acc := &Account{
		Address:         common.Address{},
		Balance:         util.NewUint128(),
		Nonce:           0,
		Staking:         util.NewUint128(),
		Voted:           voted,
		CandidateID:     nil,
		Points:          util.NewUint128(),
		LastPointsTs:    0,
		Unstaking:       util.NewUint128(),
		LastUnstakingTs: 0,
		Data:            data,
		Storage:         stor,
	}

	return acc, nil
}

func (acc *Account) fromProto(pbAcc *corepb.Account) error {
	var err error

	err = acc.Address.FromBytes(pbAcc.Address)
	if err != nil {
		return err
	}
	acc.Balance, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Balance)
	if err != nil {
		return err
	}
	acc.Nonce = pbAcc.Nonce
	acc.Staking, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Staking)
	if err != nil {
		return err
	}
	acc.Voted, err = trie.NewBatch(pbAcc.VotedRootHash, acc.Storage)
	if err != nil {
		return err
	}

	acc.CandidateID = pbAcc.CandidateId

	acc.Points, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Bandwidth)
	if err != nil {
		return err
	}
	acc.LastPointsTs = pbAcc.LastBandwidthTs

	acc.Unstaking, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Unstaking)
	if err != nil {
		return err
	}
	acc.LastUnstakingTs = pbAcc.LastUnstakingTs

	acc.Data, err = trie.NewBatch(pbAcc.DataRootHash, acc.Storage)
	if err != nil {
		return err
	}

	return nil
}

func (acc *Account) toProto() (*corepb.Account, error) {
	balance, err := acc.Balance.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	staking, err := acc.Staking.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	votedRootHash, err := acc.Voted.RootHash()
	if err != nil {
		return nil, err
	}
	bandwidth, err := acc.Points.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	unstaking, err := acc.Unstaking.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	dataRootHash, err := acc.Data.RootHash()
	if err != nil {
		return nil, err
	}

	return &corepb.Account{
		Address:         acc.Address.Bytes(),
		Balance:         balance,
		Nonce:           acc.Nonce,
		Staking:         staking,
		VotedRootHash:   votedRootHash,
		CandidateId:     acc.CandidateID,
		Bandwidth:       bandwidth,
		LastBandwidthTs: acc.LastPointsTs,
		Unstaking:       unstaking,
		LastUnstakingTs: acc.LastUnstakingTs,
		DataRootHash:    dataRootHash,
	}, nil
}

// FromBytes returns Account form bytes
func (acc *Account) FromBytes(b []byte) error {
	pbAccount := new(corepb.Account)
	if err := proto.Unmarshal(b, pbAccount); err != nil {
		return err
	}
	return acc.fromProto(pbAccount)
}

// ToBytes convert account to bytes
func (acc *Account) ToBytes() ([]byte, error) {
	pbAcc, err := acc.toProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbAcc)
}

// VotedSlice returns slice converted from Voted trie
func (acc *Account) VotedSlice() [][]byte {
	return KeyTrieToSlice(acc.Voted)
}

// GetData returns value in account's data trie
func (acc *Account) GetData(prefix string, key []byte) ([]byte, error) {
	return acc.Data.Get(append([]byte(prefix), key...))
}

// PutData put value to account's data trie
func (acc *Account) PutData(prefix string, key []byte, value []byte) error {
	return acc.Data.Put(append([]byte(prefix), key...), value)
}

// UpdatePoints update points
func (acc *Account) UpdatePoints(timestamp int64) error {
	if acc.LastPointsTs == timestamp {
		return nil
	}

	var err error

	acc.Points, err = currentPoints(acc, timestamp)
	if err != nil {
		return err
	}
	acc.LastPointsTs = timestamp

	return nil
}

// UpdateUnstaking update unstaking and balance
func (acc *Account) UpdateUnstaking(timestamp int64) error {
	var err error
	// Unstaking action does not exist
	if acc.LastUnstakingTs == 0 {
		return nil
	}

	// Staked coin is not returned if not enough time has been passed
	elapsed := timestamp - acc.LastUnstakingTs
	if time.Duration(elapsed)*time.Second < UnstakingWaitDuration {
		return nil
	}

	acc.Balance, err = acc.Balance.Add(acc.Unstaking)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add to balance.")
		return err
	}

	acc.Unstaking = util.NewUint128()
	acc.LastUnstakingTs = 0

	return nil
}
