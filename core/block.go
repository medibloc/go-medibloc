package core

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/keystore"
	"golang.org/x/crypto/sha3"
)

type BlockHeader struct {
	hash       common.Hash
	parentHash common.Hash

	coinbase  common.Address
	timestamp int64
	chainID   uint32

	alg  keystore.Algorithm
	sign []byte
}

func (b *BlockHeader) ToProto() (proto.Message, error) {
	return &corepb.BlockHeader{
		Hash:       b.hash.Bytes(),
		ParentHash: b.parentHash.Bytes(),
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
		b.coinbase = common.BytesToAddress(msg.Coinbase)
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.alg = keystore.Algorithm(msg.Alg)
		b.sign = msg.Sign
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

type Block struct {
	header       *BlockHeader
	transactions Transactions

	sealed      bool
	height      uint64
	parentBlock *Block
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
	block := &Block{
		header: &BlockHeader{

			parentHash: parent.Hash(),
			coinbase:   coinbase,
			timestamp:  time.Now().Unix(),
			chainID:    chainID,
		},
		transactions: make(Transactions, 0),
		parentBlock:  parent,
		height:       parent.height + 1,
		sealed:       false,
	}

	return block, nil
}

func (block *Block) SignThis(signature keystore.Signature) error {
	sign, err := signature.Sign(block.header.hash.Bytes())
	if err != nil {
		return err
	}
	block.header.alg = keystore.Algorithm(signature.Algorithm())
	block.header.sign = sign
	return nil
}

func (block *Block) ChainID() uint32 {
	return block.header.chainID
}

func (block *Block) Coinbase() common.Address {
	return block.header.coinbase
}

func (block *Block) Alg() keystore.Algorithm {
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

func (block *Block) Height() uint64 {
	return block.height
}

func (block *Block) Transactions() Transactions {
	return block.transactions
}

func (block *Block) Sealed() bool {
	return block.sealed
}

func (block *Block) Seal() error {
	if block.sealed {
		return ErrBlockAlreadySealed
	}
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
