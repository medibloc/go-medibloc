package transaction

import (
	"unicode"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
)

// RegisterAliasPayload is payload type for register alias
type RegisterAliasPayload struct {
	AliasName string
}

// FromBytes converts bytes to payload.
func (payload *RegisterAliasPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.RegisterAliasPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.AliasName = payloadPb.AliasName
	return nil
}

// ToBytes returns marshaled DefaultPayload
func (payload *RegisterAliasPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.RegisterAliasPayload{
		AliasName: payload.AliasName,
	}
	return proto.Marshal(payloadPb)
}

// RegisterAliasTx is a structure for register alias
type RegisterAliasTx struct {
	*core.Transaction
	addr       common.Address
	collateral *util.Uint128
	alias      string
	size       int
}

var _ core.ExecutableTx = &RegisterAliasTx{}

// NewRegisterAliasTx returns RegisterAliasTx
func NewRegisterAliasTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(RegisterAliasPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}
	if err := validateAlias(payload.AliasName); err != nil {
		return nil, err
	}

	return &RegisterAliasTx{
		Transaction: tx,
		addr:        tx.From(),
		alias:       payload.AliasName,
		collateral:  tx.Value(),
		size:        size,
	}, nil
}

// Execute RegisterAliasTx
func (tx *RegisterAliasTx) Execute(b *core.Block) error {
	collateralLimit, err := util.NewUint128FromString(AliasCollateralMinimum)
	if err != nil {
		return err
	}
	if tx.collateral.Cmp(collateralLimit) < 0 {
		return ErrAliasCollateralLimit
	}

	acc, err := b.State().GetAccount(tx.addr)
	if err != nil {
		return err
	}

	aliasBytes, err := acc.GetData("", []byte(coreState.AliasKey))
	if err != nil && err != ErrNotFound {
		return err
	}
	if aliasBytes != nil {
		return ErrAliasAlreadyHave
	}

	_, err = b.State().GetAccountByAlias(tx.alias)
	if err != nil && err != ErrNotFound {
		return err
	} else if err == nil {
		return ErrAliasAlreadyTaken
	}

	acc.Balance, err = acc.Balance.Sub(tx.collateral)
	if err == util.ErrUint128Underflow {
		return core.ErrNotEnoughBalance
	}
	if err != nil {
		return err
	}

	collateralBytes, err := tx.collateral.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}
	pbAlias := &corepb.Alias{
		AliasName:       tx.alias,
		AliasCollateral: collateralBytes,
	}
	aliasBytes, err = proto.Marshal(pbAlias)
	if err != nil {
		return err
	}
	err = acc.PutData("", []byte(coreState.AliasKey), aliasBytes)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(acc)
	if err != nil {
		return err
	}

	return b.State().PutAccountAlias(tx.alias, tx.addr)
}

// Bandwidth returns bandwidth.
func (tx *RegisterAliasTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1500, uint64(tx.size))
}

// PointChange returns account's point change when applying this transaction.
func (tx *RegisterAliasTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

// RecoverFrom returns from account's address.
func (tx *RegisterAliasTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}

// DeregisterAliasTx is a structure for deregister alias
type DeregisterAliasTx struct {
	*core.Transaction
	addr common.Address
	size int
}

var _ core.ExecutableTx = &DeregisterAliasTx{}

// NewDeregisterAliasTx returns RegisterAliasTx
func NewDeregisterAliasTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}

	return &DeregisterAliasTx{
		Transaction: tx,
		addr:        tx.From(),
		size:        size,
	}, nil
}

// Execute DeregisterAliasTx
func (tx *DeregisterAliasTx) Execute(b *core.Block) error {
	acc, err := b.State().GetAccount(tx.addr)
	if err != nil {
		return err
	}

	aliasBytes, err := acc.GetData("", []byte(coreState.AliasKey))
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == ErrNotFound || aliasBytes == nil {
		return ErrAliasNotExist
	}

	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		return err
	}
	if pbAlias.AliasName == "" {
		return ErrAliasNotExist
	}
	collateral, err := util.NewUint128FromFixedSizeByteSlice(pbAlias.AliasCollateral)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Add(collateral)
	if err != nil {
		return err
	}

	err = b.State().DelAccountAlias(pbAlias.AliasName, tx.addr)
	if err != nil {
		return err
	}

	err = acc.Data.Delete([]byte(coreState.AliasKey))
	if err != nil {
		return err
	}
	return b.State().PutAccount(acc)
}

// Bandwidth returns bandwidth.
func (tx *DeregisterAliasTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1500, uint64(tx.size))
}

// PointChange returns account's point change when applying this transaction.
func (tx *DeregisterAliasTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

// RecoverFrom returns from account's address.
func (tx *DeregisterAliasTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}

// ValidateAlias checks alias
func validateAlias(alias string) error {
	if len(alias) < AliasLengthMinimum {
		return ErrAliasLengthUnderMinimum
	}
	if len(alias) > AliasLengthMaximum {
		return ErrAliasLengthExceedMaximum
	}
	for i := 0; i < len(alias); i++ {
		ch := rune(alias[i])

		if !(unicode.IsNumber(ch) || unicode.IsLower(ch)) {
			return ErrAliasInvalidChar
		}
		if i == 0 && unicode.IsNumber(ch) {
			return ErrAliasFirstLetter
		}
	}
	return nil
}
