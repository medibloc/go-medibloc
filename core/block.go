package core

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
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
	candidacyRoot common.Hash
	consensusRoot []byte

	reservationQueueHash common.Hash

	coinbase  common.Address
	timestamp int64
	chainID   uint32

	alg  algorithm.Algorithm
	sign []byte
}

// ToProto converts BlockHeader to corepb.BlockHeader
func (b *BlockHeader) ToProto() (proto.Message, error) {
	return &corepb.BlockHeader{
		Hash:                 b.hash.Bytes(),
		ParentHash:           b.parentHash.Bytes(),
		AccsRoot:             b.accsRoot.Bytes(),
		TxsRoot:              b.txsRoot.Bytes(),
		UsageRoot:            b.usageRoot.Bytes(),
		RecordsRoot:          b.recordsRoot.Bytes(),
		CandidacyRoot:        b.candidacyRoot.Bytes(),
		ConsensusRoot:        b.consensusRoot,
		ReservationQueueHash: b.reservationQueueHash.Bytes(),
		Coinbase:             b.coinbase.Bytes(),
		Timestamp:            b.timestamp,
		ChainId:              b.chainID,
		Alg:                  uint32(b.alg),
		Sign:                 b.sign,
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
		b.candidacyRoot = common.BytesToHash(msg.CandidacyRoot)
		b.consensusRoot = msg.ConsensusRoot
		b.reservationQueueHash = common.BytesToHash(msg.ReservationQueueHash)
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
	storage   storage.Storage
	state     *BlockState
	consensus Consensus
	sealed    bool
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
	if err := block.VerifyExecution(); err != nil {
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
func (bd *BlockData) GetExecutedBlock(consensus Consensus, storage storage.Storage) (*Block, error) {
	var err error
	block := &Block{
		BlockData: bd,
		consensus: consensus,
	}
	if block.state, err = NewBlockState(block.consensus, storage); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to create new block state.")
		return nil, err
	}
	if err = block.state.LoadAccountsRoot(block.header.accsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load accounts root.")
		return nil, err
	}
	if err = block.state.LoadTransactionsRoot(block.header.txsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load transaction root.")
		return nil, err
	}
	if err = block.state.LoadUsageRoot(block.header.usageRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load usage root.")
		return nil, err
	}
	if err = block.state.LoadRecordsRoot(block.header.recordsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load records root.")
		return nil, err
	}
	if err = block.state.LoadCandidacyRoot(block.header.candidacyRoot); err != nil {
		return nil, err
	}
	if err = block.state.LoadConsensusRoot(block.consensus, block.header.consensusRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load consensus root.")
		return nil, err
	}
	if err := block.state.LoadReservationQueue(block.header.reservationQueueHash); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load reservation queue.")
		return nil, err
	}
	if err := block.state.ConstructVotesCache(); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to construct votes cache.")
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

// CandidacyRoot returns root hash of candidacy trie
func (bd *BlockData) CandidacyRoot() common.Hash {
	return bd.header.candidacyRoot
}

// ConsensusRoot returns root hash of consensus trie
func (bd *BlockData) ConsensusRoot() []byte {
	return bd.header.consensusRoot
}

// ReservationQueueHash returns hash of reservation queue
func (bd *BlockData) ReservationQueueHash() common.Hash {
	return bd.header.reservationQueueHash
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

func (bd *BlockData) String() string {
	return fmt.Sprintf("<Height:%v, Hash:%v, ParentHash:%v>", bd.Height(), bd.Hash(), bd.ParentHash())
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

	// all reserved tasks should have timestamps greater than block's timestamp
	head := block.state.PeekHeadReservedTask()
	if head != nil && head.Timestamp() < block.Timestamp() {
		return ErrReservedTaskNotProcessed
	}

	block.header.accsRoot = block.state.AccountsRoot()
	block.header.txsRoot = block.state.TransactionsRoot()
	block.header.usageRoot = block.state.UsageRoot()
	block.header.recordsRoot = block.state.RecordsRoot()
	block.header.candidacyRoot = block.state.CandidacyRoot()
	consensusRoot, err := block.state.ConsensusRoot()
	if err != nil {
		return err
	}
	block.header.consensusRoot = consensusRoot
	block.header.reservationQueueHash = block.state.ReservationQueueHash()

	hash, err := HashBlockData(block.BlockData)
	if err != nil {
		return err
	}
	block.header.hash = hash
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
	hasher.Write(bd.CandidacyRoot().Bytes())
	hasher.Write(bd.ConsensusRoot())
	hasher.Write(bd.ReservationQueueHash().Bytes())
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

	if err := block.State().TransitionDynasty(block.Timestamp()); err != nil {
		block.RollBack()
		return err
	}

	if err := block.ExecuteAll(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to execute block transactions.")
		block.RollBack()
		return err
	}

	if err := block.VerifyState(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to verify block state.")
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

	if err := block.ExecuteReservedTasks(); err != nil {
		block.RollBack()
		return err
	}

	block.Commit()

	return nil
}

// ExecuteReservedTasks processes reserved tasks with timestamp before block's timestamp
func (block *Block) ExecuteReservedTasks() error {
	tasks := block.state.PopReservedTasks(block.Timestamp())
	for _, t := range tasks {
		if err := t.ExecuteOnState(block.state); err != nil {
			return err
		}
	}
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
	if block.state.AccountsRoot() != block.AccountsRoot() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.AccountsRoot().Hex(),
			"header": block.AccountsRoot().Hex(),
		}).Warn("Failed to verify accounts root.")
		return ErrInvalidBlockAccountsRoot
	}
	if block.state.TransactionsRoot() != block.TransactionsRoot() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.TransactionsRoot().Hex(),
			"header": block.TransactionsRoot().Hex(),
		}).Warn("Failed to verify transactions root.")
		return ErrInvalidBlockTxsRoot
	}
	if block.state.UsageRoot() != block.UsageRoot() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.UsageRoot().Hex(),
			"header": block.UsageRoot().Hex(),
		}).Warn("Failed to verify usage root.")
		return ErrInvalidBlockUsageRoot
	}
	if block.state.RecordsRoot() != block.RecordsRoot() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.RecordsRoot().Hex(),
			"header": block.RecordsRoot().Hex(),
		}).Warn("Failed to verify records root.")
		return ErrInvalidBlockRecordsRoot
	}
	if block.state.CandidacyRoot() != block.CandidacyRoot() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.CandidacyRoot().Hex(),
			"header": block.CandidacyRoot().Hex(),
		}).Warn("Failed to verify candidacy root.")
		return ErrInvalidBlockCandidacyRoot
	}
	consensusRoot, err := block.state.ConsensusRoot()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get state of consensus root.")
		return err
	}
	if !byteutils.Equal(consensusRoot, block.ConsensusRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(consensusRoot),
			"header": byteutils.Bytes2Hex(block.ConsensusRoot()),
		}).Warn("Failed to verify consensus root.")
		return ErrInvalidBlockConsensusRoot
	}
	if block.state.ReservationQueueHash() != block.ReservationQueueHash() {
		logging.WithFields(logrus.Fields{
			"state":  block.state.ReservationQueueHash().Hex(),
			"header": block.ReservationQueueHash().Hex(),
		}).Warn("Failed to verify reservation queue hash.")
		return ErrInvalidBlockReservationQueueHash
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
	if bd.height == GenesisHeight {
		if !GenesisHash.Equals(bd.header.hash) {
			return ErrInvalidBlockHash
		}
		return nil
	}
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
			hash:                 block.Hash(),
			parentHash:           block.ParentHash(),
			accsRoot:             block.AccountsRoot(),
			txsRoot:              block.TransactionsRoot(),
			usageRoot:            block.UsageRoot(),
			recordsRoot:          block.RecordsRoot(),
			candidacyRoot:        block.CandidacyRoot(),
			consensusRoot:        block.ConsensusRoot(),
			reservationQueueHash: block.ReservationQueueHash(),
			coinbase:             block.Coinbase(),
			timestamp:            block.Timestamp(),
			chainID:              block.ChainID(),
			alg:                  block.Alg(),
			sign:                 block.Signature(),
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
		return nil, err
	}
	bd := new(BlockData)
	if err := bd.FromProto(pb); err != nil {
		return nil, err
	}
	return bd, nil
}
