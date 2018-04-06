package core

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/crypto/sha3"
)

type BlockHeader struct {
	hash       common.Hash
	parentHash common.Hash

	accsRoot common.Hash
	txsRoot  common.Hash

	coinbase  common.Address
	timestamp int64
	chainID   uint32

	alg  algorithm.Algorithm
	sign []byte
}

func (b *BlockHeader) ToProto() (proto.Message, error) {
	return &corepb.BlockHeader{
		Hash:       b.hash.Bytes(),
		ParentHash: b.parentHash.Bytes(),
		AccsRoot:   b.accsRoot.Bytes(),
		TxsRoot:    b.txsRoot.Bytes(),
		Coinbase:   b.coinbase.Bytes(),
		Timestamp:  b.timestamp,
		ChainId:    b.chainID,
		Alg:        uint32(b.alg),
		Sign:       b.sign,
	}, nil
}

func (b *BlockHeader) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.BlockHeader); ok {
		b.hash = common.BytesToHash(msg.Hash)
		b.parentHash = common.BytesToHash(msg.ParentHash)
		b.accsRoot = common.BytesToHash(msg.AccsRoot)
		b.txsRoot = common.BytesToHash(msg.TxsRoot)
		b.coinbase = common.BytesToAddress(msg.Coinbase)
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.alg = algorithm.Algorithm(msg.Alg)
		b.sign = msg.Sign
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

type Block struct {
	header       *BlockHeader
	transactions Transactions

	storage storage.Storage
	state   *BlockState

	sealed bool
	height uint64
}

func (block *Block) ToProto() (proto.Message, error) {
	header, err := block.header.ToProto()
	if err != nil {
		return nil, err
	}
	if header, ok := header.(*corepb.BlockHeader); ok {
		txs := make([]*corepb.Transaction, len(block.transactions))
		for idx, v := range block.transactions {
			tx, err := v.ToProto()
			if err != nil {
				return nil, err
			}
			if tx, ok := tx.(*corepb.Transaction); ok {
				txs[idx] = tx
			} else {
				return nil, ErrCannotConvertTransaction
			}
		}
		return &corepb.Block{
			Header:       header,
			Transactions: txs,
			Height:       block.height,
		}, nil
	}
	return nil, ErrInvalidBlockToProto
}

func (block *Block) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Block); ok {
		block.header = new(BlockHeader)
		if err := block.header.FromProto(msg.Header); err != nil {
			return err
		}

		block.transactions = make(Transactions, len(msg.Transactions))
		for idx, v := range msg.Transactions {
			tx := new(Transaction)
			if err := tx.FromProto(v); err != nil {
				return err
			}
			block.transactions[idx] = tx
		}
		block.height = msg.Height
		return nil
	}
	return ErrInvalidProtoToBlock
}

func NewBlock(chainID uint32, coinbase common.Address, parent *Block) (*Block, error) {
	state, err := parent.state.Clone()
	if err != nil {
		return nil, err
	}

	block := &Block{
		header: &BlockHeader{
			parentHash: parent.Hash(),
			coinbase:   coinbase,
			timestamp:  time.Now().Unix(),
			chainID:    chainID,
		},
		transactions: make(Transactions, 0),
		storage:      parent.storage,
		state:        state,
		height:       parent.height + 1,
		sealed:       false,
	}

	return block, nil
}

func (block *Block) ChainID() uint32 {
	return block.header.chainID
}

func (block *Block) Coinbase() common.Address {
	return block.header.coinbase
}

func (block *Block) Alg() algorithm.Algorithm {
	return block.header.alg
}

func (block *Block) Signature() []byte {
	return block.header.sign
}

func (block *Block) Timestamp() int64 {
	return block.header.timestamp
}

func (block *Block) SetTimestamp(timestamp int64) error {
	if block.sealed {
		return ErrBlockAlreadySealed
	}
	block.header.timestamp = timestamp
	return nil
}

func (block *Block) Hash() common.Hash {
	return block.header.hash
}

func (block *Block) ParentHash() common.Hash {
	return block.header.parentHash
}

func (block *Block) AccountsRoot() common.Hash {
	return block.header.accsRoot
}

func (block *Block) TransactionsRoot() common.Hash {
	return block.header.txsRoot
}

func (block *Block) Height() uint64 {
	return block.height
}

func (block *Block) Transactions() Transactions {
	return block.transactions
}

// TO BE REMOVED: For test without block pool
func (block *Block) SetTransactions(txs Transactions) error {
	block.transactions = txs
	return nil
}

func (block *Block) Storage() storage.Storage {
	return block.storage
}

func (block *Block) Sealed() bool {
	return block.sealed
}

func (block *Block) Seal() error {
	if block.sealed {
		return ErrBlockAlreadySealed
	}

	block.header.accsRoot = block.state.AccountsRoot()
	block.header.txsRoot = block.state.TransactionsRoot()

	var err error
	block.header.hash, err = HashBlock(block)
	if err != nil {
		return err
	}
	block.sealed = true
	return nil
}

func HashBlock(block *Block) (common.Hash, error) {
	if block == nil {
		return common.Hash{}, ErrNilArgument
	}

	hasher := sha3.New256()

	hasher.Write(block.ParentHash().Bytes())
	hasher.Write(block.header.coinbase.Bytes())
	hasher.Write(common.FromInt64(block.Timestamp()))
	hasher.Write(common.FromUint32(block.ChainID()))

	for _, tx := range block.transactions {
		hasher.Write(tx.Hash().Bytes())
	}

	return common.BytesToHash(hasher.Sum(nil)), nil
}

func (block *Block) VerifyExecution() error {
	block.BeginBatch()

	if err := block.Execute(); err != nil {
		block.RollBack()
		return err
	}

	if err := block.VerifyState(); err != nil {
		block.RollBack()
		return err
	}

	block.Commit()

	return nil
}

func (block *Block) Execute() error {
	for _, tx := range block.transactions {
		giveback, err := block.checkNonce(tx)
		if giveback {
			//TODO: return to tx pool
		}
		if err != nil {
			return err
		}

		if err = tx.Execute(block); err != nil {
			return err
		}

		if err = block.acceptTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

func (block *Block) checkNonce(tx *Transaction) (bool, error) {
	fromAcc, err := block.state.GetAccount(tx.from)
	if err != nil {
		return true, err
	}

	expectedNonce := fromAcc.Nonce() + 1
	if tx.nonce > expectedNonce {
		return true, ErrLargeTransactionNonce
	} else if tx.nonce < expectedNonce {
		return false, ErrSmallTransactionNonce
	}
	return false, nil
}

func (block *Block) acceptTransaction(tx *Transaction) error {
	pbTx, err := tx.ToProto()
	if err != nil {
		return err
	}

	txBytes, err := proto.Marshal(pbTx)
	if err != nil {
		return err
	}

	if err := block.state.PutTx(tx.hash, txBytes); err != nil {
		return err
	}

	if err = block.state.IncrementNonce(tx.from); err != nil {
		return err
	}
	return nil
}

func (block *Block) VerifyState() error {
	if !byteutils.Equal(block.state.AccountsRoot().Bytes(), block.AccountsRoot().Bytes()) {
		return ErrInvalidBlockAccountsRoot
	}
	if !byteutils.Equal(block.state.TransactionsRoot().Bytes(), block.TransactionsRoot().Bytes()) {
		return ErrInvalidBlockTxsRoot
	}
	return nil
}

func (block *Block) SignThis(signer signature.Signature) error {
	if !block.Sealed() {
		return ErrBlockNotSealed
	}

	sig, err := signer.Sign(block.header.hash.Bytes())
	if err != nil {
		return err
	}
	block.header.alg = signer.Algorithm()
	block.header.sign = sig
	return nil
}

func (block *Block) VerifyIntegrity() error {
	for _, tx := range block.transactions {
		if err := tx.VerifyIntegrity(block.header.chainID); err != nil {
			return err
		}
	}

	wantedHash, err := HashBlock(block)
	if err != nil {
		return err
	}
	if !wantedHash.Equals(block.header.hash) {
		return ErrInvalidBlockHash
	}

	// TODO: Verify according to consensus algorithm

	return nil
}

func (block *Block) BeginBatch() error {
	if err := block.state.BeginBatch(); err != nil {
		return err
	}
	return nil
}

func (block *Block) RollBack() error {
	if err := block.state.RollBack(); err != nil {
		return err
	}
	return nil
}

func (block *Block) Commit() error {
	if err := block.state.Commit(); err != nil {
		return err
	}
	return nil
}
