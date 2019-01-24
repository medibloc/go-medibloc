package transaction

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

func recoverSigner(tx *corestate.Transaction) (common.Address, error) {
	if len(tx.Sign()) == 0 {
		return common.Address{}, corestate.ErrTransactionSignatureNotExist
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

func recoverGenesis(tx *corestate.Transaction) (common.Address, error) {
	if len(tx.Sign()) != 0 {
		return common.Address{}, corestate.ErrGenesisSignShouldNotExist
	}
	return core.GenesisCoinbase, nil
}
