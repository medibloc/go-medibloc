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
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package core_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/txutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateUsage(t *testing.T) {
	coinbase := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	genesis, _, _ := testutil.NewTestGenesisBlock(t, 21)
	newBlock, err := core.NewBlock(testutil.ChainID, coinbase, genesis)
	assert.NoError(t, err)

	privHexes := []string{
		"bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0",
		"b108356a113edaaf537b6cd4f506f72787d69de3c3465adc30741653949e2173",
		"a396832a6aac41cb5844ecf86b2ec0406daa5aad4a6612964aa5e8c65abdf451",
	}

	privKeys := make([]signature.PrivateKey, len(privHexes))

	for i, privHex := range privHexes {
		privKeys[i], err = secp256k1.NewPrivateKeyFromHex(privHex)
		assert.NoError(t, err)
	}

	cases := []struct {
		from    common.Address
		privKey signature.PrivateKey
		to      common.Address
		amount  uint64
	}{
		{
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			privKeys[0],
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			10,
		},
		{
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			privKeys[0],
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			10,
		},
	}

	txs := make(core.Transactions, len(cases))
	signers := make([]signature.Signature, len(cases))

	blockState := newBlock.State()
	blockState.BeginBatch()

	for i, c := range cases {
		transaction := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(c.from).SetTo(c.to).SetValue(c.amount).
			SetType(core.TxOpSend).SetNonce(1)

		txs[i], err = core.NewTransaction(testutil.ChainID, c.from, c.to, c.amount, 1, core.TxPayloadBinaryType, []byte{})
		assert.NoError(t, err)

		signers[i], err = crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		signers[i].InitSign(c.privKey)
		assert.NoError(t, txs[i].SignThis(signers[i]))

		blockState.ExecuteTx(txs[i])

		blockState.AcceptTransaction(txs[i], newBlock.Timestamp())
	}
	blockState.Commit()

	timestamps, err := blockState.GetUsage(common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"))
	assert.NoError(t, err)
	for i, ts := range timestamps {
		assert.Equal(t, ts.Hash, txs[i].Hash())
		assert.Equal(t, ts.Timestamp, txs[i].Timestamp())
	}
}

func TestTooOldTxToAdd(t *testing.T) {
	coinbase := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	genesis, _, _ := testutil.NewTestGenesisBlock(t, 21)
	newBlock, err := core.NewBlock(testutil.ChainID, coinbase, genesis)
	assert.NoError(t, err)

	privHexes := []string{
		"bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0",
		"b108356a113edaaf537b6cd4f506f72787d69de3c3465adc30741653949e2173",
		"a396832a6aac41cb5844ecf86b2ec0406daa5aad4a6612964aa5e8c65abdf451",
	}

	privKeys := make([]signature.PrivateKey, len(privHexes))

	for i, privHex := range privHexes {
		privKeys[i], err = secp256k1.NewPrivateKeyFromHex(privHex)
		assert.NoError(t, err)
	}

	cases := []struct {
		from           common.Address
		privKey        signature.PrivateKey
		to             common.Address
		amount         *util.Uint128
		timestamp      int64
		expectedResult error
	}{
		{
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			privKeys[0],
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			util.NewUint128FromUint(10),
			time.Now().Unix(),
			nil,
		},
		{
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			privKeys[0],
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			util.NewUint128FromUint(10),
			1423276675,
			core.ErrTooOldTransaction,
		},
	}

	signers := make([]signature.Signature, len(cases))

	blockState := newBlock.State()
	blockState.BeginBatch()

	for i, c := range cases {
		tx, err := core.NewTransaction(testutil.ChainID, c.from, c.to, c.amount, 1, core.TxPayloadBinaryType, []byte{})
		assert.NoError(t, err)

		tx.SetTimestamp(c.timestamp)

		signers[i], err = crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		signers[i].InitSign(c.privKey)
		assert.NoError(t, tx.SignThis(signers[i]))

		blockState.ExecuteTx(tx)
		assert.Equal(t, c.expectedResult, blockState.AcceptTransaction(tx, newBlock.Timestamp()))
	}
}

func TestDynastyState(t *testing.T) {
	genesis, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)

	var expected []*common.Address
	for _, dynasty := range dynasties {
		expected = append(expected, &dynasty.Addr)
	}

	actual, err := dpos.DynastyStateToDynasty(genesis.State().DposState().DynastyState())
	assert.NoError(t, err)
	assert.True(t, equalSlice(expected, actual))
}

func TestCloneGenesisState(t *testing.T) {
	genesis, _, _ := testutil.NewTestGenesisBlock(t, 21)

	st := genesis.State()
	cl, err := st.Clone()
	assert.NoError(t, err)
	assert.Equal(t, st.AccountsRoot(), cl.AccountsRoot())
	assert.Equal(t, st.TransactionsRoot(), cl.TransactionsRoot())
	assert.Equal(t, st.UsageRoot(), cl.UsageRoot())
	assert.Equal(t, st.RecordsRoot(), cl.RecordsRoot())

	stDs, err := st.DposState().RootBytes()
	assert.NoError(t, err)
	clDs, err := cl.DposState().RootBytes()
	assert.NoError(t, err)
	assert.Equal(t, stDs, clDs)

	assert.Equal(t, st.ReservationQueueHash(), cl.ReservationQueueHash())
}

func TestCloneState(t *testing.T) {
	genesis, _, users := testutil.NewTestGenesisBlock(t, 21)

	st := genesis.State()

	st.BeginBatch()

	addRecordTx, err := core.NewTransaction(testutil.ChainID, users[0].Addr, common.Address{},
		util.Uint128Zero(), 1, core.TxPayloadBinaryType, []byte("abcd"))
	assert.NoError(t, err)

	assert.NoError(t, st.AddRecord(addRecordTx, []byte("recordHash"), users[0].Addr))
	assert.NoError(t, st.Vest(users[1].Addr, util.NewUint128FromUint(100)))
	assert.NoError(t, st.SubVesting(users[1].Addr, util.NewUint128FromUint(10)))
	assert.NoError(t, st.Vote(users[2].Addr, users[3].Addr))

	st.Commit()

	cl, err := st.Clone()
	assert.NoError(t, err)
	assert.Equal(t, st.AccountsRoot(), cl.AccountsRoot())
	assert.Equal(t, st.TransactionsRoot(), cl.TransactionsRoot())
	assert.Equal(t, st.UsageRoot(), cl.UsageRoot())
	assert.Equal(t, st.RecordsRoot(), cl.RecordsRoot())

	stCs, err := st.ConsensusRoot()
	assert.NoError(t, err)
	clCs, err := cl.ConsensusRoot()
	assert.NoError(t, err)
	assert.Equal(t, stCs, clCs)

	assert.Equal(t, st.CandidacyRoot(), cl.CandidacyRoot())
	assert.Equal(t, st.ReservationQueueHash(), cl.ReservationQueueHash())
}

func TestPayerUsageUpdate(t *testing.T) {
	genesis, _, users := testutil.NewTestGenesisBlock(t, 21)

	st := genesis.State()

	usage0, err := st.GetUsage(users[0].Addr)
	assert.NoError(t, err)
	usage1, err := st.GetUsage(users[1].Addr)
	assert.NoError(t, err)
	st.BeginBatch()

	txPayed, err := core.NewTransaction(testutil.ChainID, users[0].Addr, users[2].Addr,
		util.NewUint128FromUint(100), 1, core.TxPayloadBinaryType, []byte(""))
	assert.NoError(t, err)

	sig0, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig0.InitSign(users[0].PrivKey)

	sig1, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig1.InitSign(users[1].PrivKey)

	assert.NoError(t, txPayed.SignThis(sig0))
	assert.NoError(t, txPayed.SignByPayer(sig1))

	assert.NoError(t, st.ExecuteTx(txPayed))
	assert.NoError(t, st.AcceptTransaction(txPayed, txPayed.Timestamp()))

	st.Commit()

	usage0Updated, err := st.GetUsage(users[0].Addr)
	assert.NoError(t, err)
	usage1Updated, err := st.GetUsage(users[1].Addr)
	assert.NoError(t, err)

	assert.Equal(t, usage0, usage0Updated)
	assert.NotEqual(t, usage1, usage1Updated)
	assert.Equal(t, 1, len(usage1Updated))
}

func equalSlice(expected, actual []*common.Address) bool {
	if len(expected) != len(actual) {
		return false
	}
	for _, e := range expected {
		found := false
		for _, a := range actual {
			if e.Equals(*a) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
