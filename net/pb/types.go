package netpb

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// HelloMessageFromProto parse the data into Hello message
func HelloMessageFromProto(data []byte) (*Hello, error) {
	pb := new(Hello)

	if err := proto.Unmarshal(data, pb); err != nil {
		logging.VLog().WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to unmarshal Hello message.")
		return nil, err
	}

	return pb, nil
}

// OKMessageFromProto parse the data into OK message
func OKMessageFromProto(data []byte) (*OK, error) {
	pb := new(OK)

	if err := proto.Unmarshal(data, pb); err != nil {
		logging.VLog().WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to unmarshal OK message.")
		return nil, err
	}

	return pb, nil
}
