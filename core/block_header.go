package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// BlockHeader is block header
type BlockHeader struct {
	hash       []byte
	parentHash []byte

	accStateRoot []byte
	txStateRoot  []byte
	dposRoot     []byte

	coinbase  common.Address
	reward    *util.Uint128
	supply    *util.Uint128
	timestamp int64
	chainID   uint32

	sign []byte

	cpuPrice *util.Uint128
	cpuUsage uint64
	netPrice *util.Uint128
	netUsage uint64
}

// ToProto converts BlockHeader to corepb.BlockHeader
func (b *BlockHeader) ToProto() (proto.Message, error) {
	reward, err := b.reward.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	supply, err := b.supply.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	cpuPrice, err := b.cpuPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	netPrice, err := b.netPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.BlockHeader{
		Hash:         b.hash,
		ParentHash:   b.parentHash,
		Coinbase:     b.coinbase.Bytes(),
		Reward:       reward,
		Supply:       supply,
		Timestamp:    b.timestamp,
		ChainId:      b.chainID,
		Sign:         b.sign,
		AccStateRoot: b.accStateRoot,
		TxStateRoot:  b.txStateRoot,
		DposRoot:     b.dposRoot,
		CpuPrice:     cpuPrice,
		CpuUsage:     b.cpuUsage,
		NetPrice:     netPrice,
		NetUsage:     b.netUsage,
	}, nil
}

// FromProto converts corepb.BlockHeader to BlockHeader
func (b *BlockHeader) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.BlockHeader); ok {
		b.hash = msg.Hash
		b.parentHash = msg.ParentHash
		b.accStateRoot = msg.AccStateRoot
		b.txStateRoot = msg.TxStateRoot
		b.dposRoot = msg.DposRoot
		err := b.coinbase.FromBytes(msg.Coinbase)
		if err != nil {
			return err
		}
		reward, err := util.NewUint128FromFixedSizeByteSlice(msg.Reward)
		if err != nil {
			return err
		}
		b.reward = reward
		supply, err := util.NewUint128FromFixedSizeByteSlice(msg.Supply)
		if err != nil {
			return err
		}
		b.supply = supply
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.sign = msg.Sign

		cpuPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.CpuPrice)
		if err != nil {
			return err
		}

		netPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.NetPrice)
		if err != nil {
			return err
		}

		b.cpuPrice = cpuPrice
		b.cpuUsage = msg.CpuUsage
		b.netPrice = netPrice
		b.netUsage = msg.NetUsage
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

// Hash returns block header's hash
func (b *BlockHeader) Hash() []byte {
	return b.hash
}

// HexHash returns block hex encoded header's hash
func (b *BlockHeader) HexHash() string {
	return byteutils.Bytes2Hex(b.hash)
}

// SetHash set block header's hash
func (b *BlockHeader) SetHash(hash []byte) {
	b.hash = hash
}

// ParentHash returns block header's parent hash
func (b *BlockHeader) ParentHash() []byte {
	return b.parentHash
}

// SetParentHash set block header's parent hash
func (b *BlockHeader) SetParentHash(parentHash []byte) {
	b.parentHash = parentHash
}

// AccStateRoot returns block header's accStateRoot
func (b *BlockHeader) AccStateRoot() []byte {
	return b.accStateRoot
}

// SetAccStateRoot set block header's accStateRoot
func (b *BlockHeader) SetAccStateRoot(accStateRoot []byte) {
	b.accStateRoot = accStateRoot
}

// TxStateRoot returns block header's txsRoot
func (b *BlockHeader) TxStateRoot() []byte {
	return b.txStateRoot
}

// SetTxStateRoot set block header's txsRoot
func (b *BlockHeader) SetTxStateRoot(txStateRoot []byte) {
	b.txStateRoot = txStateRoot
}

// DposRoot returns block header's dposRoot
func (b *BlockHeader) DposRoot() []byte {
	return b.dposRoot
}

// SetDposRoot set block header's dposRoot
func (b *BlockHeader) SetDposRoot(dposRoot []byte) {
	b.dposRoot = dposRoot
}

// Coinbase returns coinbase
func (b *BlockHeader) Coinbase() common.Address {
	return b.coinbase
}

// SetCoinbase set coinbase
func (b *BlockHeader) SetCoinbase(coinbase common.Address) {
	b.coinbase = coinbase
}

// Reward returns reward
func (b *BlockHeader) Reward() *util.Uint128 {
	return b.reward
}

// SetReward sets reward
func (b *BlockHeader) SetReward(reward *util.Uint128) {
	b.reward = reward
}

// Supply returns supply
func (b *BlockHeader) Supply() *util.Uint128 {
	return b.supply.DeepCopy()
}

// SetSupply sets supply
func (b *BlockHeader) SetSupply(supply *util.Uint128) {
	b.supply = supply
}

// Timestamp returns timestamp of block
func (b *BlockHeader) Timestamp() int64 {
	return b.timestamp
}

// SetTimestamp sets timestamp of block
func (b *BlockHeader) SetTimestamp(timestamp int64) {
	b.timestamp = timestamp
}

// ChainID returns chainID
func (b *BlockHeader) ChainID() uint32 {
	return b.chainID
}

// SetChainID sets chainID
func (b *BlockHeader) SetChainID(chainID uint32) {
	b.chainID = chainID
}

// CPUPrice returns cpuPrice
func (b *BlockHeader) CPUPrice() *util.Uint128 {
	return b.cpuPrice
}

// SetCPUPrice sets cpuPrice
func (b *BlockHeader) SetCPUPrice(cpuPrice *util.Uint128) {
	b.cpuPrice = cpuPrice
}

// NetPrice returns netPrice
func (b *BlockHeader) NetPrice() *util.Uint128 {
	return b.netPrice
}

// SetNetPrice sets netPrice
func (b *BlockHeader) SetNetPrice(netPrice *util.Uint128) {
	b.netPrice = netPrice
}

// Sign returns sign
func (b *BlockHeader) Sign() []byte {
	return b.sign
}

// SetSign sets sign
func (b *BlockHeader) SetSign(sign []byte) {
	b.sign = sign
}

// CPUUsage returns cpuUsage
func (b *BlockHeader) CPUUsage() uint64 {
	return b.cpuUsage
}

// NetUsage returns netUsage
func (b *BlockHeader) NetUsage() uint64 {
	return b.netUsage
}

// Proposer returns proposer address from block sign
func (b *BlockHeader) Proposer() (common.Address, error) {
	if b.sign == nil {
		return common.Address{}, ErrBlockSignatureNotExist
	}
	msg := b.hash

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"algorithm": algorithm.SECP256K1,
		}).Debug("Invalid sign algorithm.")
		return common.Address{}, err
	}

	pubKey, err := sig.RecoverPublic(msg, b.sign)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":    err,
			"plain":  byteutils.Bytes2Hex(b.hash),
			"cipher": byteutils.Bytes2Hex(b.sign),
		}).Debug("Failed to recover public key from cipher text.")
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
}
