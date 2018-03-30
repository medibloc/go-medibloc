package net

import (
	"bytes"
	"errors"
	"hash/crc32"
	"time"

	byteutils "github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

/*
MedMessage defines protocol in MediBloc, we define our own wire protocol, as the following:

 0               1               2               3              (bytes)
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Magic Number                          |
+---------------------------------------------------------------+
|                         Chain ID                              |
+-----------------------------------------------+---------------+
|                         Reserved              |   Version     |
+-----------------------------------------------+---------------+
|                                                               |
+                                                               +
|                         Message Name                          |
+                                                               +
|                                                               |
+---------------------------------------------------------------+
|                         Data Length                           |
+---------------------------------------------------------------+
|                         Data Checksum                         |
+---------------------------------------------------------------+
|                         Header Checksum                       |
|---------------------------------------------------------------+
|                                                               |
+                         Data                                  +
.                                                               .
|                                                               |
+---------------------------------------------------------------+
*/
// const
const (
	MedMessageMagicNumberEndIdx    = 4
	MedMessageChainIDEndIdx        = 8
	MedMessageReservedEndIdx       = 11
	MedMessageVersionIndex         = 11
	MedMessageVersionEndIdx        = 12
	MedMessageNameEndIdx           = 24
	MedMessageDataLengthEndIdx     = 28
	MedMessageDataCheckSumEndIdx   = 32
	MedMessageHeaderCheckSumEndIdx = 36
	MedMessageHeaderLength         = 36

	// Consider that a block is too large in sync.
	MaxMedMessageDataLength = 512 * 1024 * 1024 // 512m.
	MaxMedMessageNameLength = 24 - 12           // 12.
)

// Error types
var (
	MagicNumber     = []byte{0x4e, 0x45, 0x42, 0x31}
	DefaultReserved = []byte{0x0, 0x0, 0x0}

	ErrInsufficientMessageHeaderLength = errors.New("insufficient message header length")
	ErrInsufficientMessageDataLength   = errors.New("insufficient message data length")
	ErrInvalidMagicNumber              = errors.New("invalid magic number")
	ErrInvalidHeaderCheckSum           = errors.New("invalid header checksum")
	ErrInvalidDataCheckSum             = errors.New("invalid data checksum")
	ErrExceedMaxDataLength             = errors.New("exceed max data length")
	ErrExceedMaxMessageNameLength      = errors.New("exceed max message name length")
)

//MedMessage struct
type MedMessage struct {
	content     []byte
	messageName string

	// debug fields.
	sendMessageAt  int64
	writeMessageAt int64
}

// MagicNumber return magicNumber
func (message *MedMessage) MagicNumber() []byte {
	return message.content[0:MedMessageMagicNumberEndIdx]
}

// ChainID return chainID
func (message *MedMessage) ChainID() uint32 {
	return byteutils.Uint32(message.content[MedMessageMagicNumberEndIdx:MedMessageChainIDEndIdx])
}

// Reserved return reserved
func (message *MedMessage) Reserved() []byte {
	return message.content[MedMessageChainIDEndIdx:MedMessageReservedEndIdx]
}

// Version return version
func (message *MedMessage) Version() byte {
	return message.content[MedMessageVersionIndex]
}

// MessageName return message name
func (message *MedMessage) MessageName() string {
	if message.messageName == "" {
		data := message.content[MedMessageVersionEndIdx:MedMessageNameEndIdx]
		pos := bytes.IndexByte(data, 0)
		if pos != -1 {
			message.messageName = string(data[0:pos])
		} else {
			message.messageName = string(data)
		}
	}
	return message.messageName
}

// DataLength return dataLength
func (message *MedMessage) DataLength() uint32 {
	return byteutils.Uint32(message.content[MedMessageNameEndIdx:MedMessageDataLengthEndIdx])
}

// DataCheckSum return data checkSum
func (message *MedMessage) DataCheckSum() uint32 {
	return byteutils.Uint32(message.content[MedMessageDataLengthEndIdx:MedMessageDataCheckSumEndIdx])
}

// HeaderCheckSum return header checkSum
func (message *MedMessage) HeaderCheckSum() uint32 {
	return byteutils.Uint32(message.content[MedMessageDataCheckSumEndIdx:MedMessageHeaderCheckSumEndIdx])
}

// HeaderWithoutCheckSum return header without checkSum
func (message *MedMessage) HeaderWithoutCheckSum() []byte {
	return message.content[:MedMessageDataCheckSumEndIdx]
}

// Data return data
func (message *MedMessage) Data() []byte {
	return message.content[MedMessageHeaderLength:]
}

// Content return message content
func (message *MedMessage) Content() []byte {
	return message.content
}

// Length return message Length
func (message *MedMessage) Length() uint64 {
	return uint64(len(message.content))
}

// NewMedMessage new med message
func NewMedMessage(chainID uint32, reserved []byte, version byte, messageName string, data []byte) (*MedMessage, error) {
	if len(data) > MaxMedMessageDataLength {
		logging.VLog().WithFields(logrus.Fields{
			"messageName": messageName,
			"dataLength":  len(data),
			"limits":      MaxMedMessageDataLength,
		}).Error("Exceeded max data length.")
		return nil, ErrExceedMaxDataLength
	}

	if len(messageName) > MaxMedMessageNameLength {
		logging.VLog().WithFields(logrus.Fields{
			"messageName":      messageName,
			"len(messageName)": len(messageName),
			"limits":           MaxMedMessageNameLength,
		}).Error("Exceeded max message name length.")
		return nil, ErrExceedMaxMessageNameLength

	}

	dataCheckSum := crc32.ChecksumIEEE(data)

	message := &MedMessage{
		content: make([]byte, MedMessageHeaderLength+len(data)),
	}

	// copy fields.
	copy(message.content[0:MedMessageMagicNumberEndIdx], MagicNumber)
	copy(message.content[MedMessageMagicNumberEndIdx:MedMessageChainIDEndIdx], byteutils.FromUint32(chainID))
	copy(message.content[MedMessageChainIDEndIdx:MedMessageReservedEndIdx], reserved)
	message.content[MedMessageVersionIndex] = version
	copy(message.content[MedMessageVersionEndIdx:MedMessageNameEndIdx], []byte(messageName))
	copy(message.content[MedMessageNameEndIdx:MedMessageDataLengthEndIdx], byteutils.FromUint32(uint32(len(data))))
	copy(message.content[MedMessageDataLengthEndIdx:MedMessageDataCheckSumEndIdx], byteutils.FromUint32(dataCheckSum))

	// header checksum.
	headerCheckSum := crc32.ChecksumIEEE(message.HeaderWithoutCheckSum())
	copy(message.content[MedMessageDataCheckSumEndIdx:MedMessageHeaderCheckSumEndIdx], byteutils.FromUint32(headerCheckSum))

	// copy data.
	copy(message.content[MedMessageHeaderCheckSumEndIdx:], data)

	return message, nil
}

// ParseMedMessage parse med message
func ParseMedMessage(data []byte) (*MedMessage, error) {
	if len(data) < MedMessageHeaderLength {
		return nil, ErrInsufficientMessageHeaderLength
	}

	message := &MedMessage{
		content: make([]byte, MedMessageHeaderLength),
	}
	copy(message.content, data)

	if err := message.VerifyHeader(); err != nil {
		return nil, err
	}

	return message, nil
}

// ParseMessageData parse med message data
func (message *MedMessage) ParseMessageData(data []byte) error {
	if uint32(len(data)) < message.DataLength() {
		return ErrInsufficientMessageDataLength
	}

	message.content = append(message.content, data[:message.DataLength()]...)
	return message.VerifyData()
}

// VerifyHeader verify message header
func (message *MedMessage) VerifyHeader() error {
	if !byteutils.Equal(MagicNumber, message.MagicNumber()) {
		logging.VLog().WithFields(logrus.Fields{
			"expect": MagicNumber,
			"actual": message.MagicNumber(),
			"err":    "invalid magic number",
		}).Debug("Failed to verify header.")
		return ErrInvalidMagicNumber
	}

	expectedCheckSum := crc32.ChecksumIEEE(message.HeaderWithoutCheckSum())
	if expectedCheckSum != message.HeaderCheckSum() {
		logging.VLog().WithFields(logrus.Fields{
			"expect": expectedCheckSum,
			"actual": message.HeaderCheckSum(),
			"err":    "invalid header checksum",
		}).Debug("Failed to verify header.")
		return ErrInvalidHeaderCheckSum
	}

	if message.DataLength() > MaxMedMessageDataLength {
		logging.VLog().WithFields(logrus.Fields{
			"messageName": message.MessageName(),
			"dataLength":  message.DataLength(),
			"limit":       MaxMedMessageDataLength,
			"err":         "exceeded max data length",
		}).Debug("Failed to verify header.")
		return ErrExceedMaxDataLength
	}

	return nil
}

// VerifyData verify message data
func (message *MedMessage) VerifyData() error {
	expectedCheckSum := crc32.ChecksumIEEE(message.Data())
	if expectedCheckSum != message.DataCheckSum() {
		logging.VLog().WithFields(logrus.Fields{
			"expect": expectedCheckSum,
			"actual": message.DataCheckSum(),
			"err":    "invalid data checksum",
		}).Debug("Failed to verify data")
		return ErrInvalidDataCheckSum
	}
	return nil
}

// FlagSendMessageAt flag of send message time
func (message *MedMessage) FlagSendMessageAt() {
	message.sendMessageAt = time.Now().UnixNano()
}

// FlagWriteMessageAt flag of write message time
func (message *MedMessage) FlagWriteMessageAt() {
	message.writeMessageAt = time.Now().UnixNano()
}

// LatencyFromSendToWrite latency from sendMessage to writeMessage
func (message *MedMessage) LatencyFromSendToWrite() int64 {
	if message.sendMessageAt == 0 {
		return -1
	} else if message.writeMessageAt == 0 {
		message.FlagWriteMessageAt()
	}

	// convert from nano to millisecond.
	return (message.writeMessageAt - message.sendMessageAt) / int64(time.Millisecond)
}
