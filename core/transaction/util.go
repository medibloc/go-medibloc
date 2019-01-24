package transaction

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

func recoverSigner(tx *core.Transaction) (common.Address, error) {
	if len(tx.Sign()) == 0 {
		return common.Address{}, core.ErrTransactionSignatureNotExist
	}

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	if err != nil {
		return common.Address{}, err
	}

	pubKey, err := sig.RecoverPublic(tx.Hash(), tx.Sign())
	if err != nil {
		return common.Address{}, err
	}
	return common.PublicKeyToAddress(pubKey)
}

func recoverGenesis(tx *core.Transaction) (common.Address, error) {
	if len(tx.Sign()) != 0 {
		return common.Address{}, core.ErrGenesisSignShouldNotExist
	}
	return core.GenesisCoinbase, nil
}
