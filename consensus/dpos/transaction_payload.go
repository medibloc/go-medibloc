package dpos

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
)

// BecomeCandidatePayload is payload type for BecomeCandidate
type BecomeCandidatePayload struct {
	URL string
}

// FromBytes converts bytes to payload.
func (payload *BecomeCandidatePayload) FromBytes(b []byte) error {
	payloadPb := &dpospb.BecomeCandidatePayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.URL = payloadPb.Url
	return nil
}

// ToBytes returns marshaled BecomeCandidatePayload
func (payload *BecomeCandidatePayload) ToBytes() ([]byte, error) {
	payloadPb := &dpospb.BecomeCandidatePayload{
		Url: payload.URL,
	}
	return proto.Marshal(payloadPb)
}

// VotePayload is payload type for VoteTx
type VotePayload struct {
	CandidateIDs [][]byte
}

// FromBytes converts bytes to payload.
func (payload *VotePayload) FromBytes(b []byte) error {
	payloadPb := new(dpospb.VotePayload)
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.CandidateIDs = payloadPb.CandidateIDs
	return nil
}

// ToBytes returns marshaled RevokeCertificationPayload
func (payload *VotePayload) ToBytes() ([]byte, error) {
	payloadPb := &dpospb.VotePayload{
		CandidateIDs: payload.CandidateIDs,
	}
	return proto.Marshal(payloadPb)
}
