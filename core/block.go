package core

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	byteutils "github.com/medibloc/go-medibloc/util/bytes"
	"github.com/medibloc/go-medibloc/util/logging"
	"golang.org/x/crypto/sha3"
)

// BlockHeader is block header
type BlockHeader struct {
	hash       common.Hash
	parentHash common.Hash

	accsRoot      common.Hash
	txsRoot       common.Hash
	usageRoot     common.Hash
	recordsRoot   common.Hash
	consensusRoot common.Hash

	coinbase  common.Address
	timestamp int64
	chainID   uint32

	alg  algorithm.Algorithm
	sign []byte
}

// ToProto converts BlockHeader to corepb.BlockHeader
func (b *BlockHeader) ToProto() (proto.Message, error) {
	return &corepb.BlockHeader{
		Hash:          b.hash.Bytes(),
		ParentHash:    b.parentHash.Bytes(),
		AccsRoot:      b.accsRoot.Bytes(),
		TxsRoot:       b.txsRoot.Bytes(),
		UsageRoot:     b.usageRoot.Bytes(),
		RecordsRoot:   b.recordsRoot.Bytes(),
		ConsensusRoot: b.consensusRoot.Bytes(),
		Coinbase:      b.coinbase.Bytes(),
		Timestamp:     b.timestamp,
		ChainId:       b.chainID,
		Alg:           uint32(b.alg),
		Sign:          b.sign,
	}, nil
}

// FromProto converts corepb.BlockHeader to BlockHeader
func (b *BlockHeader) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.BlockHeader); ok {
		b.hash = common.BytesToHash(msg.Hash)
		b.parentHash = common.BytesToHash(msg.ParentHash)
		b.accsRoot = common.BytesToHash(msg.AccsRoot)
		b.txsRoot = common.BytesToHash(msg.TxsRoot)
		b.usageRoot = common.BytesToHash(msg.UsageRoot)
		b.recordsRoot = common.BytesToHash(msg.RecordsRoot)
		b.consensusRoot = common.BytesToHash(msg.ConsensusRoot)
		b.coinbase = common.BytesToAddress(msg.Coinbase)
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.alg = algorithm.Algorithm(msg.Alg)
		b.sign = msg.Sign
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

// BlockData represents a block
type BlockData struct {
	header       *BlockHeader
	transactions Transactions
	height       uint64
}

// Block represents block with actual state tries
type Block struct {
	*BlockData
	storage storage.Storage
	state   *BlockState
	sealed  bool
}

// ToProto converts Block to corepb.Block
func (bd *BlockData) ToProto() (proto.Message, error) {
	header, err := bd.header.ToProto()
	if err != nil {
		return nil, err
	}
	if header, ok := header.(*corepb.BlockHeader); ok {
		txs := make([]*corepb.Transaction, len(bd.transactions))
		for idx, v := range bd.transactions {
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
			Height:       bd.height,
		}, nil
	}
	return nil, ErrInvalidBlockToProto
}

// FromProto converts corepb.Block to Block
func (bd *BlockData) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Block); ok {
		bd.header = new(BlockHeader)
		if err := bd.header.FromProto(msg.Header); err != nil {
			return err
		}

		bd.transactions = make(Transactions, len(msg.Transactions))
		for idx, v := range msg.Transactions {
			tx := new(Transaction)
			if err := tx.FromProto(v); err != nil {
				return err
			}
			bd.transactions[idx] = tx
		}
		bd.height = msg.Height
		return nil
	}
	return ErrInvalidProtoToBlock
}

// NewBlock initialize new block data
func NewBlock(chainID uint32, coinbase common.Address, parent *Block) (*Block, error) {
	state, err := parent.state.Clone()
	if err != nil {
		return nil, err
	}

	block := &Block{
		BlockData: &BlockData{
			header: &BlockHeader{
				parentHash: parent.Hash(),
				coinbase:   coinbase,
				timestamp:  time.Now().Unix(),
				chainID:    chainID,
			},
			transactions: make(Transactions, 0),
			height:       parent.height + 1,
		},
		storage: parent.storage,
		state:   state,
		sealed:  false,
	}

	return block, nil
}

// ExecuteOnParentBlock returns Block object with state after block execution
func (bd *BlockData) ExecuteOnParentBlock(parent *Block) (*Block, error) {
	block, err := prepareExecution(bd, parent)
	if err != nil {
		return nil, err
	}
	if err := block.ExecuteAll(); err != nil {
		return nil, err
	}
	return block, err
}

// prepareExecution by setting states and storage as those of parents
func prepareExecution(bd *BlockData, parent *Block) (*Block, error) {
	var err error
	block := &Block{
		BlockData: bd,
	}
	if block.state, err = parent.state.Clone(); err != nil {
		return nil, err
	}
	block.storage = parent.storage
	return block, nil
}

// GetExecutedBlock converts BlockData instance to an already executed Block instance
func (bd *BlockData) GetExecutedBlock(storage storage.Storage) (*Block, error) {
	var err error
	block := &Block{
		BlockData: bd,
	}
	if block.state, err = NewBlockState(storage); err != nil {
		return nil, err
	}
	if err = block.state.LoadAccountsRoot(block.header.accsRoot); err != nil {
		return nil, err
	}
	if err = block.state.LoadTransactionsRoot(block.header.txsRoot); err != nil {
		return nil, err
	}
	if common.IsZeroHash(block.header.usageRoot) == false {
		if err = block.state.LoadUsageRoot(block.header.usageRoot); err != nil {
			return nil, err
		}
	}
	if common.IsZeroHash(block.header.recordsRoot) == false {
		if err = block.state.LoadRecordsRoot(block.header.recordsRoot); err != nil {
			return nil, err
		}
	}
	if err = block.state.LoadConsensusRoot(block.header.consensusRoot); err != nil {
		return nil, err
	}
	block.storage = storage
	return block, nil
}

// ChainID returns chain id
func (bd *BlockData) ChainID() uint32 {
	return bd.header.chainID
}

// Coinbase returns coinbase address
func (bd *BlockData) Coinbase() common.Address {
	return bd.header.coinbase
}

// Alg returns sign algorithm
func (bd *BlockData) Alg() algorithm.Algorithm {
	return bd.header.alg
}

// Signature returns block signature
func (bd *BlockData) Signature() []byte {
	return bd.header.sign
}

// Timestamp returns timestamp
func (bd *BlockData) Timestamp() int64 {
	return bd.header.timestamp
}

// SetTimestamp set block timestamp
func (bd *BlockData) SetTimestamp(timestamp int64) error {
	bd.header.timestamp = timestamp
	return nil
}

// Hash returns block hash
func (bd *BlockData) Hash() common.Hash {
	return bd.header.hash
}

// ParentHash returns hash of parent block
func (bd *BlockData) ParentHash() common.Hash {
	return bd.header.parentHash
}

// State returns block state
func (block *Block) State() *BlockState {
	return block.state
}

// AccountsRoot returns root hash of accounts trie
func (bd *BlockData) AccountsRoot() common.Hash {
	return bd.header.accsRoot
}

// TransactionsRoot returns root hash of txs trie
func (bd *BlockData) TransactionsRoot() common.Hash {
	return bd.header.txsRoot
}

// UsageRoot returns root hash of usage trie
func (bd *BlockData) UsageRoot() common.Hash {
	return bd.header.usageRoot
}

// RecordsRoot returns root hash of records trie
func (bd *BlockData) RecordsRoot() common.Hash {
	return bd.header.recordsRoot
}

// ConsensusRoot returns root hash of consensus trie
func (bd *BlockData) ConsensusRoot() common.Hash {
	return bd.header.consensusRoot
}

// Height returns height
func (bd *BlockData) Height() uint64 {
	return bd.height
}

// Transactions returns txs in block
func (bd *BlockData) Transactions() Transactions {
	return bd.transactions
}

// SetTransactions sets transactions TO BE REMOVED: For test without block pool
func (bd *BlockData) SetTransactions(txs Transactions) error {
	bd.transactions = txs
	return nil
}

// Storage returns storage used by block
func (block *Block) Storage() storage.Storage {
	return block.storage
}

// Sealed returns sealed
func (block *Block) Sealed() bool {
	return block.sealed
}

// Seal writes state root hashes and block hash in block header
func (block *Block) Seal() error {
	if block.sealed {
		return ErrBlockAlreadySealed
	}

	block.header.accsRoot = block.state.AccountsRoot()
	block.header.txsRoot = block.state.TransactionsRoot()
	block.header.usageRoot = block.state.UsageRoot()
	block.header.recordsRoot = block.state.RecordsRoot()
	block.header.consensusRoot = block.state.ConsensusRoot()

	var err error
	block.header.hash, err = HashBlockData(block.BlockData)
	if err != nil {
		return err
	}
	block.sealed = true
	return nil
}

// HashBlockData returns hash of block
func HashBlockData(bd *BlockData) (common.Hash, error) {
	if bd == nil {
		return common.Hash{}, ErrNilArgument
	}

	hasher := sha3.New256()

	hasher.Write(bd.ParentHash().Bytes())
	hasher.Write(bd.Coinbase().Bytes())
	hasher.Write(bd.AccountsRoot().Bytes())
	hasher.Write(bd.TransactionsRoot().Bytes())
	hasher.Write(bd.UsageRoot().Bytes())
	hasher.Write(bd.RecordsRoot().Bytes())
	hasher.Write(bd.ConsensusRoot().Bytes())
	hasher.Write(byteutils.FromInt64(bd.Timestamp()))
	hasher.Write(byteutils.FromUint32(bd.ChainID()))

	for _, tx := range bd.transactions {
		hasher.Write(tx.Hash().Bytes())
	}

	return common.BytesToHash(hasher.Sum(nil)), nil
}

// ExecuteTransaction on given block state
func (block *Block) ExecuteTransaction(tx *Transaction) error {
	return block.state.ExecuteTx(tx)
}

// VerifyExecution executes txs in block and verify root hashes using block header
func (block *Block) VerifyExecution() error {
	block.BeginBatch()

	if err := block.ExecuteAll(); err != nil {
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

// ExecuteAll executes all txs in block
func (block *Block) ExecuteAll() error {
	block.BeginBatch()

	for _, tx := range block.transactions {
		if err := block.ExecuteTransaction(tx); err != nil {
			block.RollBack()
			return err
		}

		if err := block.state.AcceptTransaction(tx, block.Timestamp()); err != nil {
			block.RollBack()
			return err
		}
	}

	block.Commit()

	return nil
}

// AcceptTransaction adds tx in block state
func (block *Block) AcceptTransaction(tx *Transaction) error {
	if err := block.state.AcceptTransaction(tx, block.Timestamp()); err != nil {
		return err
	}
	block.transactions = append(block.transactions, tx)
	return nil
}

// VerifyState verifies block states comparing with root hashes in header
func (block *Block) VerifyState() error {
	if !byteutils.Equal(block.state.AccountsRoot().Bytes(), block.AccountsRoot().Bytes()) {
		return ErrInvalidBlockAccountsRoot
	}
	if !byteutils.Equal(block.state.TransactionsRoot().Bytes(), block.TransactionsRoot().Bytes()) {
		return ErrInvalidBlockTxsRoot
	}
	if !byteutils.Equal(block.state.UsageRoot().Bytes(), block.UsageRoot().Bytes()) {
		return ErrInvalidBlockTxsRoot
	}
	if !byteutils.Equal(block.state.RecordsRoot().Bytes(), block.RecordsRoot().Bytes()) {
		return ErrInvalidBlockTxsRoot
	}
	if !byteutils.Equal(block.state.ConsensusRoot().Bytes(), block.ConsensusRoot().Bytes()) {
		return ErrInvalidBlockConsensusRoot
	}
	return nil
}

// SignThis sets signature info in block data
func (bd *BlockData) SignThis(signer signature.Signature) error {
	sig, err := signer.Sign(bd.header.hash.Bytes())
	if err != nil {
		return err
	}
	bd.header.alg = signer.Algorithm()
	bd.header.sign = sig
	return nil
}

// SignThis sets signature info in block
func (block *Block) SignThis(signer signature.Signature) error {
	if !block.Sealed() {
		return ErrBlockNotSealed
	}

	return block.BlockData.SignThis(signer)
}

// VerifyIntegrity verifies if block signature is valid
func (bd *BlockData) VerifyIntegrity() error {
	for _, tx := range bd.transactions {
		if err := tx.VerifyIntegrity(bd.header.chainID); err != nil {
			return err
		}
	}

	wantedHash, err := HashBlockData(bd)
	if err != nil {
		return err
	}
	if !wantedHash.Equals(bd.header.hash) {
		return ErrInvalidBlockHash
	}

	// TODO: Verify according to consensus algorithm

	return nil
}

// BeginBatch makes block state update possible
func (block *Block) BeginBatch() error {
	return block.state.BeginBatch()
}

// RollBack rolls back block state batch updates
func (block *Block) RollBack() error {
	return block.state.RollBack()
}

// Commit saves batch updates to storage
func (block *Block) Commit() error {
	return block.state.Commit()
}

// GetBlockData returns data part of block
func (block *Block) GetBlockData() *BlockData {
	bd := &BlockData{
		header: &BlockHeader{
			hash:          block.Hash(),
			parentHash:    block.ParentHash(),
			accsRoot:      block.AccountsRoot(),
			txsRoot:       block.TransactionsRoot(),
			usageRoot:     block.UsageRoot(),
			recordsRoot:   block.RecordsRoot(),
			consensusRoot: block.ConsensusRoot(),
			coinbase:      block.Coinbase(),
			timestamp:     block.Timestamp(),
			chainID:       block.ChainID(),
			alg:           block.Alg(),
			sign:          block.Signature(),
		},
		height: block.Height(),
	}

	txs := make(Transactions, len(block.transactions))
	for i, t := range block.transactions {
		txs[i] = t
	}
	bd.transactions = txs

	return bd
}

func bytesToBlockData(bytes []byte) (*BlockData, error) {
	pb := new(corepb.Block)
	if err := proto.Unmarshal(bytes, pb); err != nil {
		logging.Debug("") // TODO
		return nil, err
	}
	bd := new(BlockData)
	if err := bd.FromProto(pb); err != nil {
		logging.Debug("") // TODO
		return nil, err
	}
	return bd, nil
}
