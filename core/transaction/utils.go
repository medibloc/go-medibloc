package transaction

import "github.com/medibloc/go-medibloc/util/byteutils"

// BytesToTransactionPayload convert byte slice to Payload
func BytesToTransactionPayload(bytes []byte, payload Payload) error {
	if err := payload.FromBytes(bytes); err != nil {
		return ErrFailedToUnmarshalPayload
	}
	b, err := payload.ToBytes()
	if err != nil {
		return ErrFailedToMarshalPayload
	}
	if !byteutils.Equal(bytes, b) {
		return ErrCheckPayloadIntegrity
	}
	return nil
}
