package byteutils

// Encoder is encoder for byteutils.Encode().
type Encoder interface {
	EncodeToBytes(s interface{}) ([]byte, error)
}

// Decoder is decoder for byteutils.Decode().
type Decoder interface {
	DecodeFromBytes(data []byte) (interface{}, error)
}
