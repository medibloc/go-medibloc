package crypto

import (
  "crypto/aes"
  "crypto/cipher"
)

func AESCTRXOR(key, inText, iv []byte) ([]byte, error) {
  // AES-128 is selected due to size of encryptKey.
  aesBlock, err := aes.NewCipher(key)
  if err != nil {
    return nil, err
  }
  stream := cipher.NewCTR(aesBlock, iv)
  outText := make([]byte, len(inText))
  stream.XORKeyStream(outText, inText)
  return outText, nil
}
