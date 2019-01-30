package blockutil

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
)

func init() {
	core.InjectTxMapper(transaction.DefaultTxMap)
}

var (
	// ChainID is chain id for test configuration.
	ChainID uint32 = 1

	// DynastySize is dynasty size for test configuration
	DynastySize = 3
)
