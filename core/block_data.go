package core

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/event"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// BlockData represents a block
type BlockData struct {
	*BlockHeader
	transactions []*Transaction
	height       uint64
}

// ToProto converts Block to corepb.Block
func (bd *BlockData) ToProto() (proto.Message, error) {
	header, err := bd.BlockHeader.toProto()
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
				return nil, err
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
		bd.BlockHeader = new(BlockHeader)
		if err := bd.BlockHeader.fromProto(msg.Header); err != nil {
			return err
		}

		bd.transactions = make([]*Transaction, len(msg.Transactions))
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

// ToBytes convert block data to byte slice
func (bd *BlockData) ToBytes() ([]byte, error) {
	pb, err := bd.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pb)
}

// FromBytes convert byte slice to
func (bd *BlockData) FromBytes(bytes []byte) error {
	pb := new(corepb.Block)
	if err := proto.Unmarshal(bytes, pb); err != nil {
		return err
	}
	if err := bd.FromProto(pb); err != nil {
		return err
	}
	return nil
}

// Clone copy block data
func (bd *BlockData) Clone() (*BlockData, error) {
	protoBd, err := bd.ToProto()
	if err != nil {
		return nil, err
	}
	newBd := new(BlockData)
	err = newBd.FromProto(protoBd)
	if err != nil {
		return nil, err
	}
	return newBd, nil
}

// Height returns height
func (bd *BlockData) Height() uint64 {
	return bd.height
}

// Transactions returns txs in block
func (bd *BlockData) Transactions() []*Transaction {
	return bd.transactions
}

// AppendTransaction append transaction to block data (only use on making block)
func (bd *BlockData) AppendTransaction(tx *Transaction) {
	bd.transactions = append(bd.transactions, tx)
}

// String implements Stringer interface.
func (bd *BlockData) String() string {
	proposer, _ := bd.Proposer()
	return fmt.Sprintf("<height:%v, hash:%v, parent_hash:%v, coinbase:%v, reward:%v, supply:%v, timestamp:%v, "+
		"proposer:%v, cpuPrice:%v, cpuBandwidth:%v, netPrice:%v, netBandwidth:%v>",
		bd.Height(),
		byteutils.Bytes2Hex(bd.Hash()),
		byteutils.Bytes2Hex(bd.ParentHash()),
		byteutils.Bytes2Hex(bd.Coinbase().Bytes()),
		bd.Reward().String(),
		bd.Supply().String(),
		bd.Timestamp(),
		proposer.Hex(),
		bd.CPUPrice(),
		bd.CPUUsage(),
		bd.NetPrice(),
		bd.NetUsage(),
	)
}

// SignThis sets signature info in block data
func (bd *BlockData) SignThis(signer signature.Signature) error {
	sig, err := signer.Sign(bd.hash)
	if err != nil {
		return err
	}
	bd.sign = sig
	return nil
}

// VerifyIntegrity verifies block integrity.
func (bd *BlockData) VerifyIntegrity() error {
	// TODO Verify integrity of genesis block
	if bd.height == GenesisHeight {
		return nil
	}
	for _, tx := range bd.transactions {
		if err := tx.VerifyIntegrity(bd.chainID); err != nil {
			return err
		}
	}

	if err := bd.verifyUsageData(); err != nil {
		return err
	}

	wantedHash, err := bd.CalcHash()
	if err != nil {
		return err
	}
	if !byteutils.Equal(wantedHash, bd.hash) {
		return ErrInvalidBlockHash
	}

	return nil
}

// CalcHash returns hash of block
func (bd *BlockData) CalcHash() ([]byte, error) {
	rewardBytes, err := bd.Reward().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	supplyBytes, err := bd.Supply().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	cpuPriceBytes, err := bd.CPUPrice().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	netPriceBytes, err := bd.NetPrice().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	txHash := make([][]byte, len(bd.transactions))
	for _, tx := range bd.transactions {
		txHash = append(txHash, tx.Hash())
	}

	blockHashTarget := &corepb.BlockHashTarget{
		ParentHash:   bd.ParentHash(),
		Coinbase:     bd.Coinbase().Bytes(),
		AccStateRoot: bd.AccStateRoot(),
		TxStateRoot:  bd.TxStateRoot(),
		DposRoot:     bd.DposRoot(),
		Timestamp:    bd.Timestamp(),
		ChainId:      bd.ChainID(),
		Reward:       rewardBytes,
		Supply:       supplyBytes,
		CpuPrice:     cpuPriceBytes,
		CpuUsage:     bd.cpuUsage,
		NetPrice:     netPriceBytes,
		NetUsage:     bd.netUsage,
		TxHash:       txHash,
	}
	blockHashTargetBytes, err := proto.Marshal(blockHashTarget)
	if err != nil {
		return nil, err
	}

	return hash.GenHash(algorithm.SHA3256, blockHashTargetBytes)
}

func (bd *BlockData) verifyUsageData() error {
	if err := bd.verifyCPUUsage(); err != nil {
		return err
	}
	if err := bd.verifyNetUsage(); err != nil {
		return err
	}
	return nil
}

func (bd *BlockData) verifyCPUUsage() error {
	if bd.CPUUsage() > CPULimit {
		return ErrInvalidCPUUsage
	}

	cpuUsage := uint64(0)
	for _, tx := range bd.Transactions() {
		cpuUsage = cpuUsage + tx.Receipt().CPUUsage()
	}

	if cpuUsage != bd.cpuUsage {
		return ErrWrongCPUUsage
	}
	return nil
}

func (bd *BlockData) verifyNetUsage() error {
	if bd.NetUsage() > NetLimit {
		return ErrInvalidNetUsage
	}

	netUsage := uint64(0)
	for _, tx := range bd.Transactions() {
		netUsage = netUsage + tx.Receipt().NetUsage()
	}

	if netUsage != bd.netUsage {
		return ErrWrongNetUsage
	}
	return nil
}

// EmitTxExecutionEvent emits events of txs in the block
func (bd *BlockData) EmitTxExecutionEvent(emitter *event.Emitter) {
	for _, tx := range bd.Transactions() {
		tx.TriggerEvent(emitter, event.TopicTransactionExecutionResult)
		tx.TriggerAccEvent(emitter, event.TypeAccountTransactionExecution)
	}
}

// EmitBlockEvent emits block related event
func (bd *BlockData) EmitBlockEvent(emitter *event.Emitter, eTopic string) {
	ev := &event.Event{
		Topic: eTopic,
		Data:  byteutils.Bytes2Hex(bd.Hash()),
		Type:  "",
	}
	emitter.Trigger(ev)
}
