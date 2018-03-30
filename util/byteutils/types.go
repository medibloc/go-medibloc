package byteutils

// Encoder is encoder for bytes.Encode().
type Encoder interface {
	EncodeToBytes(s interface{}) ([]byte, error)
}

// Decoder is decoder for bytes.Decode().
type Decoder interface {
	DecodeFromBytes(data []byte) (interface{}, error)
}
