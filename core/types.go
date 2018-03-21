package core

import (
  "error"
)

var (
  ErrCannotConvertTransaction = errors.New("proto message cannot be converted into Transaction")
)
