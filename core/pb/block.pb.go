// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: block.proto

/*
Package corepb is a generated protocol buffer package.

It is generated from these files:
	block.proto

It has these top-level messages:
	BlockHeader
	Block
	DownloadParentBlock
	BlockHashTarget
	Transaction
	Receipt
	TransactionHashTarget
	TransactionPayerSignTarget
	DefaultPayload
	AddCertificationPayload
	RevokeCertificationPayload
	AddRecordPayload
	RegisterAliasPayload
	Genesis
	GenesisMeta
	GenesisTokenDistribution
*/
package corepb

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type BlockHeader struct {
	Hash         []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	ParentHash   []byte `protobuf:"bytes,2,opt,name=parent_hash,json=parentHash,proto3" json:"parent_hash,omitempty"`
	Coinbase     []byte `protobuf:"bytes,3,opt,name=coinbase,proto3" json:"coinbase,omitempty"`
	Reward       []byte `protobuf:"bytes,4,opt,name=reward,proto3" json:"reward,omitempty"`
	Supply       []byte `protobuf:"bytes,5,opt,name=supply,proto3" json:"supply,omitempty"`
	Timestamp    int64  `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	ChainId      uint32 `protobuf:"varint,7,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	HashAlg      uint32 `protobuf:"varint,10,opt,name=hash_alg,json=hashAlg,proto3" json:"hash_alg,omitempty"`
	CryptoAlg    uint32 `protobuf:"varint,11,opt,name=crypto_alg,json=cryptoAlg,proto3" json:"crypto_alg,omitempty"`
	Sign         []byte `protobuf:"bytes,12,opt,name=sign,proto3" json:"sign,omitempty"`
	AccStateRoot []byte `protobuf:"bytes,21,opt,name=acc_state_root,json=accStateRoot,proto3" json:"acc_state_root,omitempty"`
	TxStateRoot  []byte `protobuf:"bytes,22,opt,name=tx_state_root,json=txStateRoot,proto3" json:"tx_state_root,omitempty"`
	DposRoot     []byte `protobuf:"bytes,23,opt,name=dpos_root,json=dposRoot,proto3" json:"dpos_root,omitempty"`
	CpuRef       []byte `protobuf:"bytes,30,opt,name=cpu_ref,json=cpuRef,proto3" json:"cpu_ref,omitempty"`
	CpuUsage     []byte `protobuf:"bytes,31,opt,name=cpu_usage,json=cpuUsage,proto3" json:"cpu_usage,omitempty"`
	NetRef       []byte `protobuf:"bytes,32,opt,name=net_ref,json=netRef,proto3" json:"net_ref,omitempty"`
	NetUsage     []byte `protobuf:"bytes,33,opt,name=net_usage,json=netUsage,proto3" json:"net_usage,omitempty"`
}

func (m *BlockHeader) Reset()                    { *m = BlockHeader{} }
func (m *BlockHeader) String() string            { return proto.CompactTextString(m) }
func (*BlockHeader) ProtoMessage()               {}
func (*BlockHeader) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{0} }

func (m *BlockHeader) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *BlockHeader) GetParentHash() []byte {
	if m != nil {
		return m.ParentHash
	}
	return nil
}

func (m *BlockHeader) GetCoinbase() []byte {
	if m != nil {
		return m.Coinbase
	}
	return nil
}

func (m *BlockHeader) GetReward() []byte {
	if m != nil {
		return m.Reward
	}
	return nil
}

func (m *BlockHeader) GetSupply() []byte {
	if m != nil {
		return m.Supply
	}
	return nil
}

func (m *BlockHeader) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *BlockHeader) GetChainId() uint32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *BlockHeader) GetHashAlg() uint32 {
	if m != nil {
		return m.HashAlg
	}
	return 0
}

func (m *BlockHeader) GetCryptoAlg() uint32 {
	if m != nil {
		return m.CryptoAlg
	}
	return 0
}

func (m *BlockHeader) GetSign() []byte {
	if m != nil {
		return m.Sign
	}
	return nil
}

func (m *BlockHeader) GetAccStateRoot() []byte {
	if m != nil {
		return m.AccStateRoot
	}
	return nil
}

func (m *BlockHeader) GetTxStateRoot() []byte {
	if m != nil {
		return m.TxStateRoot
	}
	return nil
}

func (m *BlockHeader) GetDposRoot() []byte {
	if m != nil {
		return m.DposRoot
	}
	return nil
}

func (m *BlockHeader) GetCpuRef() []byte {
	if m != nil {
		return m.CpuRef
	}
	return nil
}

func (m *BlockHeader) GetCpuUsage() []byte {
	if m != nil {
		return m.CpuUsage
	}
	return nil
}

func (m *BlockHeader) GetNetRef() []byte {
	if m != nil {
		return m.NetRef
	}
	return nil
}

func (m *BlockHeader) GetNetUsage() []byte {
	if m != nil {
		return m.NetUsage
	}
	return nil
}

type Block struct {
	Header       *BlockHeader   `protobuf:"bytes,1,opt,name=header" json:"header,omitempty"`
	Transactions []*Transaction `protobuf:"bytes,2,rep,name=transactions" json:"transactions,omitempty"`
	Height       uint64         `protobuf:"varint,3,opt,name=height,proto3" json:"height,omitempty"`
}

func (m *Block) Reset()                    { *m = Block{} }
func (m *Block) String() string            { return proto.CompactTextString(m) }
func (*Block) ProtoMessage()               {}
func (*Block) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{1} }

func (m *Block) GetHeader() *BlockHeader {
	if m != nil {
		return m.Header
	}
	return nil
}

func (m *Block) GetTransactions() []*Transaction {
	if m != nil {
		return m.Transactions
	}
	return nil
}

func (m *Block) GetHeight() uint64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type DownloadParentBlock struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Sign []byte `protobuf:"bytes,2,opt,name=sign,proto3" json:"sign,omitempty"`
}

func (m *DownloadParentBlock) Reset()                    { *m = DownloadParentBlock{} }
func (m *DownloadParentBlock) String() string            { return proto.CompactTextString(m) }
func (*DownloadParentBlock) ProtoMessage()               {}
func (*DownloadParentBlock) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{2} }

func (m *DownloadParentBlock) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *DownloadParentBlock) GetSign() []byte {
	if m != nil {
		return m.Sign
	}
	return nil
}

type BlockHashTarget struct {
	ParentHash   []byte   `protobuf:"bytes,1,opt,name=parent_hash,json=parentHash,proto3" json:"parent_hash,omitempty"`
	Coinbase     []byte   `protobuf:"bytes,2,opt,name=coinbase,proto3" json:"coinbase,omitempty"`
	AccStateRoot []byte   `protobuf:"bytes,3,opt,name=acc_state_root,json=accStateRoot,proto3" json:"acc_state_root,omitempty"`
	TxStateRoot  []byte   `protobuf:"bytes,4,opt,name=tx_state_root,json=txStateRoot,proto3" json:"tx_state_root,omitempty"`
	DposRoot     []byte   `protobuf:"bytes,5,opt,name=dpos_root,json=dposRoot,proto3" json:"dpos_root,omitempty"`
	Timestamp    int64    `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	ChainId      uint32   `protobuf:"varint,7,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Reward       []byte   `protobuf:"bytes,8,opt,name=reward,proto3" json:"reward,omitempty"`
	Supply       []byte   `protobuf:"bytes,9,opt,name=supply,proto3" json:"supply,omitempty"`
	HashAlg      uint32   `protobuf:"varint,10,opt,name=hash_alg,json=hashAlg,proto3" json:"hash_alg,omitempty"`
	CryptoAlg    uint32   `protobuf:"varint,11,opt,name=crypto_alg,json=cryptoAlg,proto3" json:"crypto_alg,omitempty"`
	CpuUsage     []byte   `protobuf:"bytes,20,opt,name=cpu_usage,json=cpuUsage,proto3" json:"cpu_usage,omitempty"`
	NetUsage     []byte   `protobuf:"bytes,21,opt,name=net_usage,json=netUsage,proto3" json:"net_usage,omitempty"`
	TxHash       [][]byte `protobuf:"bytes,30,rep,name=tx_hash,json=txHash" json:"tx_hash,omitempty"`
}

func (m *BlockHashTarget) Reset()                    { *m = BlockHashTarget{} }
func (m *BlockHashTarget) String() string            { return proto.CompactTextString(m) }
func (*BlockHashTarget) ProtoMessage()               {}
func (*BlockHashTarget) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{3} }

func (m *BlockHashTarget) GetParentHash() []byte {
	if m != nil {
		return m.ParentHash
	}
	return nil
}

func (m *BlockHashTarget) GetCoinbase() []byte {
	if m != nil {
		return m.Coinbase
	}
	return nil
}

func (m *BlockHashTarget) GetAccStateRoot() []byte {
	if m != nil {
		return m.AccStateRoot
	}
	return nil
}

func (m *BlockHashTarget) GetTxStateRoot() []byte {
	if m != nil {
		return m.TxStateRoot
	}
	return nil
}

func (m *BlockHashTarget) GetDposRoot() []byte {
	if m != nil {
		return m.DposRoot
	}
	return nil
}

func (m *BlockHashTarget) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *BlockHashTarget) GetChainId() uint32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *BlockHashTarget) GetReward() []byte {
	if m != nil {
		return m.Reward
	}
	return nil
}

func (m *BlockHashTarget) GetSupply() []byte {
	if m != nil {
		return m.Supply
	}
	return nil
}

func (m *BlockHashTarget) GetHashAlg() uint32 {
	if m != nil {
		return m.HashAlg
	}
	return 0
}

func (m *BlockHashTarget) GetCryptoAlg() uint32 {
	if m != nil {
		return m.CryptoAlg
	}
	return 0
}

func (m *BlockHashTarget) GetCpuUsage() []byte {
	if m != nil {
		return m.CpuUsage
	}
	return nil
}

func (m *BlockHashTarget) GetNetUsage() []byte {
	if m != nil {
		return m.NetUsage
	}
	return nil
}

func (m *BlockHashTarget) GetTxHash() [][]byte {
	if m != nil {
		return m.TxHash
	}
	return nil
}

type Transaction struct {
	Hash      []byte   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	TxType    string   `protobuf:"bytes,2,opt,name=tx_type,json=txType,proto3" json:"tx_type,omitempty"`
	To        []byte   `protobuf:"bytes,3,opt,name=to,proto3" json:"to,omitempty"`
	Value     []byte   `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp int64    `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Nonce     uint64   `protobuf:"varint,6,opt,name=nonce,proto3" json:"nonce,omitempty"`
	ChainId   uint32   `protobuf:"varint,7,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Payload   []byte   `protobuf:"bytes,10,opt,name=payload,proto3" json:"payload,omitempty"`
	HashAlg   uint32   `protobuf:"varint,20,opt,name=hash_alg,json=hashAlg,proto3" json:"hash_alg,omitempty"`
	CryptoAlg uint32   `protobuf:"varint,21,opt,name=crypto_alg,json=cryptoAlg,proto3" json:"crypto_alg,omitempty"`
	Sign      []byte   `protobuf:"bytes,22,opt,name=sign,proto3" json:"sign,omitempty"`
	PayerSign []byte   `protobuf:"bytes,23,opt,name=payerSign,proto3" json:"payerSign,omitempty"`
	Receipt   *Receipt `protobuf:"bytes,30,opt,name=receipt" json:"receipt,omitempty"`
}

func (m *Transaction) Reset()                    { *m = Transaction{} }
func (m *Transaction) String() string            { return proto.CompactTextString(m) }
func (*Transaction) ProtoMessage()               {}
func (*Transaction) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{4} }

func (m *Transaction) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *Transaction) GetTxType() string {
	if m != nil {
		return m.TxType
	}
	return ""
}

func (m *Transaction) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *Transaction) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Transaction) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Transaction) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Transaction) GetChainId() uint32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *Transaction) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *Transaction) GetHashAlg() uint32 {
	if m != nil {
		return m.HashAlg
	}
	return 0
}

func (m *Transaction) GetCryptoAlg() uint32 {
	if m != nil {
		return m.CryptoAlg
	}
	return 0
}

func (m *Transaction) GetSign() []byte {
	if m != nil {
		return m.Sign
	}
	return nil
}

func (m *Transaction) GetPayerSign() []byte {
	if m != nil {
		return m.PayerSign
	}
	return nil
}

func (m *Transaction) GetReceipt() *Receipt {
	if m != nil {
		return m.Receipt
	}
	return nil
}

type Receipt struct {
	Executed bool   `protobuf:"varint,1,opt,name=executed,proto3" json:"executed,omitempty"`
	CpuUsage []byte `protobuf:"bytes,2,opt,name=cpu_usage,json=cpuUsage,proto3" json:"cpu_usage,omitempty"`
	NetUsage []byte `protobuf:"bytes,3,opt,name=net_usage,json=netUsage,proto3" json:"net_usage,omitempty"`
	Error    []byte `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
}

func (m *Receipt) Reset()                    { *m = Receipt{} }
func (m *Receipt) String() string            { return proto.CompactTextString(m) }
func (*Receipt) ProtoMessage()               {}
func (*Receipt) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{5} }

func (m *Receipt) GetExecuted() bool {
	if m != nil {
		return m.Executed
	}
	return false
}

func (m *Receipt) GetCpuUsage() []byte {
	if m != nil {
		return m.CpuUsage
	}
	return nil
}

func (m *Receipt) GetNetUsage() []byte {
	if m != nil {
		return m.NetUsage
	}
	return nil
}

func (m *Receipt) GetError() []byte {
	if m != nil {
		return m.Error
	}
	return nil
}

type TransactionHashTarget struct {
	TxType    string `protobuf:"bytes,1,opt,name=tx_type,json=txType,proto3" json:"tx_type,omitempty"`
	From      []byte `protobuf:"bytes,2,opt,name=from,proto3" json:"from,omitempty"`
	To        []byte `protobuf:"bytes,3,opt,name=to,proto3" json:"to,omitempty"`
	Value     []byte `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp int64  `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Nonce     uint64 `protobuf:"varint,6,opt,name=nonce,proto3" json:"nonce,omitempty"`
	ChainId   uint32 `protobuf:"varint,7,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Payload   []byte `protobuf:"bytes,10,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *TransactionHashTarget) Reset()                    { *m = TransactionHashTarget{} }
func (m *TransactionHashTarget) String() string            { return proto.CompactTextString(m) }
func (*TransactionHashTarget) ProtoMessage()               {}
func (*TransactionHashTarget) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{6} }

func (m *TransactionHashTarget) GetTxType() string {
	if m != nil {
		return m.TxType
	}
	return ""
}

func (m *TransactionHashTarget) GetFrom() []byte {
	if m != nil {
		return m.From
	}
	return nil
}

func (m *TransactionHashTarget) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *TransactionHashTarget) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *TransactionHashTarget) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *TransactionHashTarget) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *TransactionHashTarget) GetChainId() uint32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *TransactionHashTarget) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type TransactionPayerSignTarget struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Sign []byte `protobuf:"bytes,2,opt,name=sign,proto3" json:"sign,omitempty"`
}

func (m *TransactionPayerSignTarget) Reset()                    { *m = TransactionPayerSignTarget{} }
func (m *TransactionPayerSignTarget) String() string            { return proto.CompactTextString(m) }
func (*TransactionPayerSignTarget) ProtoMessage()               {}
func (*TransactionPayerSignTarget) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{7} }

func (m *TransactionPayerSignTarget) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *TransactionPayerSignTarget) GetSign() []byte {
	if m != nil {
		return m.Sign
	}
	return nil
}

type DefaultPayload struct {
	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *DefaultPayload) Reset()                    { *m = DefaultPayload{} }
func (m *DefaultPayload) String() string            { return proto.CompactTextString(m) }
func (*DefaultPayload) ProtoMessage()               {}
func (*DefaultPayload) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{8} }

func (m *DefaultPayload) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

type AddCertificationPayload struct {
	IssueTime      int64  `protobuf:"varint,1,opt,name=issue_time,json=issueTime,proto3" json:"issue_time,omitempty"`
	ExpirationTime int64  `protobuf:"varint,2,opt,name=expiration_time,json=expirationTime,proto3" json:"expiration_time,omitempty"`
	Hash           []byte `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *AddCertificationPayload) Reset()                    { *m = AddCertificationPayload{} }
func (m *AddCertificationPayload) String() string            { return proto.CompactTextString(m) }
func (*AddCertificationPayload) ProtoMessage()               {}
func (*AddCertificationPayload) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{9} }

func (m *AddCertificationPayload) GetIssueTime() int64 {
	if m != nil {
		return m.IssueTime
	}
	return 0
}

func (m *AddCertificationPayload) GetExpirationTime() int64 {
	if m != nil {
		return m.ExpirationTime
	}
	return 0
}

func (m *AddCertificationPayload) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type RevokeCertificationPayload struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *RevokeCertificationPayload) Reset()                    { *m = RevokeCertificationPayload{} }
func (m *RevokeCertificationPayload) String() string            { return proto.CompactTextString(m) }
func (*RevokeCertificationPayload) ProtoMessage()               {}
func (*RevokeCertificationPayload) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{10} }

func (m *RevokeCertificationPayload) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type AddRecordPayload struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *AddRecordPayload) Reset()                    { *m = AddRecordPayload{} }
func (m *AddRecordPayload) String() string            { return proto.CompactTextString(m) }
func (*AddRecordPayload) ProtoMessage()               {}
func (*AddRecordPayload) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{11} }

func (m *AddRecordPayload) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type RegisterAliasPayload struct {
	AliasName string `protobuf:"bytes,1,opt,name=alias_name,json=aliasName,proto3" json:"alias_name,omitempty"`
}

func (m *RegisterAliasPayload) Reset()                    { *m = RegisterAliasPayload{} }
func (m *RegisterAliasPayload) String() string            { return proto.CompactTextString(m) }
func (*RegisterAliasPayload) ProtoMessage()               {}
func (*RegisterAliasPayload) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{12} }

func (m *RegisterAliasPayload) GetAliasName() string {
	if m != nil {
		return m.AliasName
	}
	return ""
}

type Genesis struct {
	// genesis meta
	Meta *GenesisMeta `protobuf:"bytes,1,opt,name=meta" json:"meta,omitempty"`
	// genesis token distribution address
	TokenDistribution []*GenesisTokenDistribution `protobuf:"bytes,2,rep,name=token_distribution,json=tokenDistribution" json:"token_distribution,omitempty"`
	// genesis transactions
	Transactions []*Transaction `protobuf:"bytes,3,rep,name=transactions" json:"transactions,omitempty"`
}

func (m *Genesis) Reset()                    { *m = Genesis{} }
func (m *Genesis) String() string            { return proto.CompactTextString(m) }
func (*Genesis) ProtoMessage()               {}
func (*Genesis) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{13} }

func (m *Genesis) GetMeta() *GenesisMeta {
	if m != nil {
		return m.Meta
	}
	return nil
}

func (m *Genesis) GetTokenDistribution() []*GenesisTokenDistribution {
	if m != nil {
		return m.TokenDistribution
	}
	return nil
}

func (m *Genesis) GetTransactions() []*Transaction {
	if m != nil {
		return m.Transactions
	}
	return nil
}

type GenesisMeta struct {
	// ChainID.
	ChainId uint32 `protobuf:"varint,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	// Dynasty size.
	DynastySize uint32 `protobuf:"varint,2,opt,name=dynasty_size,json=dynastySize,proto3" json:"dynasty_size,omitempty"`
}

func (m *GenesisMeta) Reset()                    { *m = GenesisMeta{} }
func (m *GenesisMeta) String() string            { return proto.CompactTextString(m) }
func (*GenesisMeta) ProtoMessage()               {}
func (*GenesisMeta) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{14} }

func (m *GenesisMeta) GetChainId() uint32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *GenesisMeta) GetDynastySize() uint32 {
	if m != nil {
		return m.DynastySize
	}
	return 0
}

type GenesisTokenDistribution struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Balance string `protobuf:"bytes,2,opt,name=balance,proto3" json:"balance,omitempty"`
}

func (m *GenesisTokenDistribution) Reset()                    { *m = GenesisTokenDistribution{} }
func (m *GenesisTokenDistribution) String() string            { return proto.CompactTextString(m) }
func (*GenesisTokenDistribution) ProtoMessage()               {}
func (*GenesisTokenDistribution) Descriptor() ([]byte, []int) { return fileDescriptorBlock, []int{15} }

func (m *GenesisTokenDistribution) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *GenesisTokenDistribution) GetBalance() string {
	if m != nil {
		return m.Balance
	}
	return ""
}

func init() {
	proto.RegisterType((*BlockHeader)(nil), "corepb.BlockHeader")
	proto.RegisterType((*Block)(nil), "corepb.Block")
	proto.RegisterType((*DownloadParentBlock)(nil), "corepb.DownloadParentBlock")
	proto.RegisterType((*BlockHashTarget)(nil), "corepb.BlockHashTarget")
	proto.RegisterType((*Transaction)(nil), "corepb.Transaction")
	proto.RegisterType((*Receipt)(nil), "corepb.Receipt")
	proto.RegisterType((*TransactionHashTarget)(nil), "corepb.TransactionHashTarget")
	proto.RegisterType((*TransactionPayerSignTarget)(nil), "corepb.TransactionPayerSignTarget")
	proto.RegisterType((*DefaultPayload)(nil), "corepb.DefaultPayload")
	proto.RegisterType((*AddCertificationPayload)(nil), "corepb.AddCertificationPayload")
	proto.RegisterType((*RevokeCertificationPayload)(nil), "corepb.RevokeCertificationPayload")
	proto.RegisterType((*AddRecordPayload)(nil), "corepb.AddRecordPayload")
	proto.RegisterType((*RegisterAliasPayload)(nil), "corepb.RegisterAliasPayload")
	proto.RegisterType((*Genesis)(nil), "corepb.Genesis")
	proto.RegisterType((*GenesisMeta)(nil), "corepb.GenesisMeta")
	proto.RegisterType((*GenesisTokenDistribution)(nil), "corepb.GenesisTokenDistribution")
}

func init() { proto.RegisterFile("block.proto", fileDescriptorBlock) }

var fileDescriptorBlock = []byte{
	// 957 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x56, 0xdd, 0x6e, 0xe3, 0x44,
	0x14, 0x96, 0xe3, 0xfc, 0x34, 0xc7, 0x69, 0x0b, 0xb3, 0x69, 0x6b, 0xca, 0xfe, 0x64, 0x2d, 0xc4,
	0x16, 0x90, 0x2a, 0xb4, 0x08, 0x71, 0xc5, 0x45, 0xa1, 0x12, 0x8b, 0x10, 0x4b, 0x35, 0x0d, 0xd7,
	0xd6, 0xc4, 0x3e, 0x49, 0x46, 0x4d, 0x3c, 0xd6, 0xcc, 0x78, 0x37, 0xd9, 0x7b, 0x9e, 0x86, 0x57,
	0xe0, 0x2d, 0x78, 0x01, 0x2e, 0x79, 0x0c, 0x34, 0x33, 0xf6, 0xc6, 0xce, 0xb6, 0xa1, 0xda, 0xab,
	0xbd, 0xf3, 0xf9, 0xbe, 0x73, 0x94, 0x99, 0xef, 0xfb, 0x66, 0x26, 0x10, 0x4c, 0x16, 0x22, 0xb9,
	0x39, 0xcf, 0xa5, 0xd0, 0x82, 0x74, 0x13, 0x21, 0x31, 0x9f, 0x44, 0xff, 0xfa, 0x10, 0xfc, 0x60,
	0xf0, 0x17, 0xc8, 0x52, 0x94, 0x84, 0x40, 0x7b, 0xce, 0xd4, 0x3c, 0xf4, 0x46, 0xde, 0xd9, 0x80,
	0xda, 0x6f, 0xf2, 0x04, 0x82, 0x9c, 0x49, 0xcc, 0x74, 0x6c, 0xa9, 0x96, 0xa5, 0xc0, 0x41, 0x2f,
	0x4c, 0xc3, 0x29, 0xec, 0x25, 0x82, 0x67, 0x13, 0xa6, 0x30, 0xf4, 0x2d, 0xfb, 0xb6, 0x26, 0xc7,
	0xd0, 0x95, 0xf8, 0x9a, 0xc9, 0x34, 0x6c, 0x5b, 0xa6, 0xac, 0x0c, 0xae, 0x8a, 0x3c, 0x5f, 0xac,
	0xc3, 0x8e, 0xc3, 0x5d, 0x45, 0x1e, 0x42, 0x5f, 0xf3, 0x25, 0x2a, 0xcd, 0x96, 0x79, 0xd8, 0x1d,
	0x79, 0x67, 0x3e, 0xdd, 0x00, 0xe4, 0x13, 0xd8, 0x4b, 0xe6, 0x8c, 0x67, 0x31, 0x4f, 0xc3, 0xde,
	0xc8, 0x3b, 0xdb, 0xa7, 0x3d, 0x5b, 0xff, 0x9c, 0x1a, 0xca, 0x2c, 0x2f, 0x66, 0x8b, 0x59, 0x08,
	0x8e, 0x32, 0xf5, 0xc5, 0x62, 0x46, 0x1e, 0x01, 0x24, 0x72, 0x9d, 0x6b, 0x61, 0xc9, 0xc0, 0x92,
	0x7d, 0x87, 0x18, 0x9a, 0x40, 0x5b, 0xf1, 0x59, 0x16, 0x0e, 0xdc, 0x9e, 0xcd, 0x37, 0xf9, 0x0c,
	0x0e, 0x58, 0x92, 0xc4, 0x4a, 0x33, 0x8d, 0xb1, 0x14, 0x42, 0x87, 0x47, 0x96, 0x1d, 0xb0, 0x24,
	0xb9, 0x36, 0x20, 0x15, 0x42, 0x93, 0x08, 0xf6, 0xf5, 0xaa, 0xde, 0x74, 0x6c, 0x9b, 0x02, 0xbd,
	0xda, 0xf4, 0x7c, 0x0a, 0xfd, 0x34, 0x17, 0xca, 0xf1, 0x27, 0x4e, 0x1d, 0x03, 0x58, 0xf2, 0x04,
	0x7a, 0x49, 0x5e, 0xc4, 0x12, 0xa7, 0xe1, 0x63, 0x27, 0x43, 0x92, 0x17, 0x14, 0xa7, 0x66, 0xca,
	0x10, 0x85, 0x62, 0x33, 0x0c, 0x9f, 0x94, 0x9a, 0xe6, 0xc5, 0xef, 0xa6, 0x36, 0x53, 0x19, 0x6a,
	0x3b, 0x35, 0x72, 0x53, 0x19, 0xea, 0x72, 0xca, 0x10, 0x6e, 0xea, 0xa9, 0x9b, 0xca, 0x50, 0xdb,
	0xa9, 0xe8, 0x0f, 0x0f, 0x3a, 0xd6, 0x6a, 0xf2, 0x15, 0x74, 0xe7, 0xd6, 0x6e, 0x6b, 0x73, 0xf0,
	0xfc, 0xc1, 0xb9, 0x4b, 0xc3, 0x79, 0x2d, 0x09, 0xb4, 0x6c, 0x21, 0xdf, 0xc1, 0x40, 0x4b, 0x96,
	0x29, 0x96, 0x68, 0x2e, 0x32, 0x15, 0xb6, 0x46, 0x7e, 0x7d, 0x64, 0xbc, 0xe1, 0x68, 0xa3, 0xd1,
	0x38, 0x3c, 0x47, 0x3e, 0x9b, 0x6b, 0x9b, 0x89, 0x36, 0x2d, 0xab, 0xe8, 0x7b, 0x78, 0x70, 0x29,
	0x5e, 0x67, 0x0b, 0xc1, 0xd2, 0x2b, 0x9b, 0x21, 0xb7, 0xa8, 0xdb, 0x92, 0x57, 0x39, 0xd3, 0xda,
	0x38, 0x13, 0xfd, 0xe9, 0xc3, 0xa1, 0x5b, 0x27, 0x53, 0xf3, 0x31, 0x93, 0x33, 0xd4, 0xdb, 0x09,
	0xf5, 0x76, 0x26, 0xb4, 0xb5, 0x95, 0xd0, 0x77, 0xad, 0xf6, 0xef, 0x63, 0x75, 0xfb, 0x7f, 0xac,
	0xee, 0x6c, 0x59, 0xfd, 0xde, 0xc1, 0xde, 0x9c, 0xa0, 0xbd, 0x3b, 0x4e, 0x50, 0xbf, 0x71, 0x82,
	0xde, 0xff, 0x20, 0x34, 0x42, 0x37, 0xdc, 0x0a, 0x5d, 0x23, 0x5b, 0x47, 0xcd, 0x6c, 0x99, 0x44,
	0xea, 0x95, 0x13, 0xff, 0xf1, 0xc8, 0x37, 0x8b, 0xd1, 0x2b, 0x23, 0x7c, 0xf4, 0x4f, 0x0b, 0x82,
	0x5a, 0x44, 0x6e, 0x75, 0xd9, 0x0d, 0xeb, 0x75, 0xee, 0xbc, 0xe9, 0x9b, 0xe1, 0xf1, 0x3a, 0x47,
	0x72, 0x00, 0x2d, 0x2d, 0x4a, 0x37, 0x5a, 0x5a, 0x90, 0x21, 0x74, 0x5e, 0xb1, 0x45, 0x81, 0xa5,
	0xf6, 0xae, 0x68, 0x0a, 0xdb, 0xd9, 0x16, 0x76, 0x08, 0x9d, 0x4c, 0x64, 0x09, 0x5a, 0xc9, 0xdb,
	0xd4, 0x15, 0xbb, 0xe4, 0x0e, 0xa1, 0x97, 0xb3, 0xb5, 0x49, 0xa7, 0x55, 0x6f, 0x40, 0xab, 0xb2,
	0x21, 0xec, 0x70, 0x97, 0xb0, 0x47, 0x77, 0xdd, 0x30, 0xc7, 0xb5, 0x1b, 0xe6, 0x21, 0xf4, 0x73,
	0xb6, 0x46, 0x79, 0x6d, 0x08, 0x77, 0x2f, 0x6c, 0x00, 0xf2, 0x05, 0xf4, 0x24, 0x26, 0xc8, 0x73,
	0x6d, 0x2f, 0x86, 0xe0, 0xf9, 0x61, 0x75, 0xe0, 0xa8, 0x83, 0x69, 0xc5, 0x47, 0x05, 0xf4, 0x4a,
	0xcc, 0xc4, 0x1c, 0x57, 0x98, 0x14, 0x1a, 0x53, 0xab, 0xf0, 0x1e, 0x7d, 0x5b, 0x37, 0xcd, 0x6d,
	0xed, 0x32, 0xd7, 0xdf, 0x32, 0x77, 0x08, 0x1d, 0x94, 0x52, 0xc8, 0x4a, 0x76, 0x5b, 0x44, 0x7f,
	0x7b, 0x70, 0x54, 0x73, 0xb6, 0x76, 0x1a, 0x6b, 0x7e, 0x7a, 0x0d, 0x3f, 0x09, 0xb4, 0xa7, 0x52,
	0x2c, 0xab, 0xe3, 0x6c, 0xbe, 0x3f, 0x30, 0x8f, 0xa3, 0x4b, 0x38, 0xad, 0x6d, 0xea, 0xaa, 0xf2,
	0xa3, 0xdc, 0xd9, 0x7d, 0xef, 0xa8, 0x2f, 0xe1, 0xe0, 0x12, 0xa7, 0xac, 0x58, 0xe8, 0xab, 0x32,
	0x3b, 0x21, 0xf4, 0x96, 0xa8, 0xac, 0xbc, 0x4e, 0x93, 0xaa, 0x8c, 0x0a, 0x38, 0xb9, 0x48, 0xd3,
	0x1f, 0x51, 0x6a, 0x3e, 0xe5, 0x09, 0x2b, 0x7f, 0xd6, 0x0e, 0x3d, 0x02, 0xe0, 0x4a, 0x15, 0x18,
	0x9b, 0xad, 0xda, 0x39, 0x9f, 0xf6, 0x2d, 0x32, 0xe6, 0x4b, 0x24, 0xcf, 0xe0, 0x10, 0x57, 0x39,
	0x97, 0x76, 0xc6, 0xf5, 0xb4, 0x6c, 0xcf, 0xc1, 0x06, 0xb6, 0x8d, 0xd5, 0xb2, 0xfd, 0xcd, 0xb2,
	0xa3, 0xaf, 0xe1, 0x94, 0xe2, 0x2b, 0x71, 0x83, 0xb7, 0xfe, 0xf2, 0x2d, 0x1b, 0x8d, 0x3e, 0x87,
	0x8f, 0x2e, 0xd2, 0x94, 0x62, 0x22, 0x64, 0xba, 0xab, 0xef, 0x5b, 0x18, 0x52, 0x9c, 0x71, 0xa5,
	0x51, 0x5e, 0x2c, 0x38, 0x53, 0xb5, 0xdd, 0x30, 0x53, 0xc7, 0x19, 0x5b, 0x56, 0x2a, 0xf4, 0x2d,
	0xf2, 0x92, 0x2d, 0x31, 0xfa, 0xcb, 0x83, 0xde, 0x4f, 0x98, 0xa1, 0xe2, 0x8a, 0x3c, 0x83, 0xf6,
	0x12, 0x35, 0xdb, 0x7e, 0x9e, 0x4a, 0xfa, 0x57, 0xd4, 0x8c, 0xda, 0x06, 0xf2, 0x1b, 0x10, 0x2d,
	0x6e, 0x30, 0x8b, 0x53, 0xae, 0xb4, 0xe4, 0x93, 0xc2, 0x6c, 0xa2, 0x7c, 0xa2, 0x46, 0x5b, 0x63,
	0x63, 0xd3, 0x78, 0x59, 0xeb, 0xa3, 0x1f, 0xeb, 0x6d, 0xe8, 0x9d, 0xd7, 0xce, 0xbf, 0xe7, 0x6b,
	0x17, 0xfd, 0x02, 0x41, 0x6d, 0x79, 0x8d, 0xf0, 0x79, 0xcd, 0xf0, 0x3d, 0x85, 0x41, 0xba, 0xce,
	0x98, 0xd2, 0xeb, 0x58, 0xf1, 0x37, 0xce, 0xb3, 0x7d, 0x1a, 0x94, 0xd8, 0x35, 0x7f, 0x83, 0xd1,
	0x4b, 0x08, 0xef, 0x5a, 0xb4, 0x49, 0x12, 0x4b, 0x53, 0x89, 0x4a, 0x55, 0x49, 0x2a, 0x4b, 0xc3,
	0x4c, 0xd8, 0x82, 0x99, 0x83, 0xe0, 0xee, 0xd1, 0xaa, 0x9c, 0x74, 0xed, 0x9f, 0xbe, 0x6f, 0xfe,
	0x0b, 0x00, 0x00, 0xff, 0xff, 0x1b, 0xd6, 0x81, 0x59, 0x03, 0x0a, 0x00, 0x00,
}
