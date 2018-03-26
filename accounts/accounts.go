package accounts

import (
  "github.com/medibloc/go-medibloc/common"
)

type Account struct {
  Address common.Address `json:"address"`
  URL     URL            `json:"url"`
}
