package rpc

import (
	"encoding/json"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func generatePayloadBuf(txData *rpcpb.TransactionData) ([]byte, error) {
	var addRecord *core.AddRecordPayload
	var addCertification *core.AddCertificationPayload
	var revokeCertification *core.RevokeCertificationPayload

	switch txData.Type {
	case core.TxOpTransfer:
		return nil, nil
	case core.TxOpAddRecord:
		json.Unmarshal([]byte(txData.Payload), &addRecord)
		payload := core.NewAddRecordPayload(addRecord.Hash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpVest:
		return nil, nil
	case core.TxOpWithdrawVesting:
		return nil, nil
	case core.TxOpAddCertification:
		json.Unmarshal([]byte(txData.Payload), &addCertification)
		payload := core.NewAddCertificationPayload(addCertification.IssueTime,
			addCertification.ExpirationTime, addCertification.CertificateHash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpRevokeCertification:
		json.Unmarshal([]byte(txData.Payload), &revokeCertification)
		payload := core.NewRevokeCertificationPayload(revokeCertification.CertificateHash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case dpos.TxOpBecomeCandidate:
		return nil, nil
	case dpos.TxOpQuitCandidacy:
		return nil, nil
	case dpos.TxOpVote:
		return nil, nil
	}
	return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidDataType)
}
