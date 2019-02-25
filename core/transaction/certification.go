package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// AddCertificationPayload is payload type for AddCertificationTx
type AddCertificationPayload struct {
	IssueTime       int64
	ExpirationTime  int64
	CertificateHash []byte
}

// FromBytes converts bytes to payload.
func (payload *AddCertificationPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.AddCertificationPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.IssueTime = payloadPb.IssueTime
	payload.ExpirationTime = payloadPb.ExpirationTime
	payload.CertificateHash = payloadPb.Hash
	return nil
}

// ToBytes returns marshaled AddCertificationPayload
func (payload *AddCertificationPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.AddCertificationPayload{
		IssueTime:      payload.IssueTime,
		ExpirationTime: payload.ExpirationTime,
		Hash:           payload.CertificateHash,
	}
	return proto.Marshal(payloadPb)
}

// AddCertificationTx is a structure for adding certification
type AddCertificationTx struct {
	*core.Transaction
	Issuer          common.Address
	Certified       common.Address
	CertificateHash []byte
	IssueTime       int64
	ExpirationTime  int64
	size            int
}

var _ core.ExecutableTx = &AddCertificationTx{}

// NewAddCertificationTx returns AddCertificationTx
func NewAddCertificationTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(AddCertificationPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) || !common.IsHexAddress(tx.To().Hex()) {
		return nil, ErrInvalidAddress
	}
	if !common.IsHash(byteutils.Bytes2Hex(payload.CertificateHash)) {
		return nil, ErrCertHashInvalid
	}

	return &AddCertificationTx{
		Transaction:     tx,
		Issuer:          tx.From(),
		Certified:       tx.To(),
		CertificateHash: payload.CertificateHash,
		IssueTime:       payload.IssueTime,
		ExpirationTime:  payload.ExpirationTime,
		size:            size,
	}, nil
}

// Execute AddCertificationTx
func (tx *AddCertificationTx) Execute(b *core.Block) error {
	certified, err := b.State().GetAccount(tx.Certified)
	if err != nil {
		return err
	}
	_, err = certified.GetData(coreState.CertReceivedPrefix, tx.CertificateHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertReceivedAlreadyAdded
	}

	issuer, err := b.State().GetAccount(tx.Issuer)
	if err != nil {
		return err
	}
	_, err = issuer.GetData(coreState.CertIssuedPrefix, tx.CertificateHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertIssuedAlreadyAdded
	}

	// TODO: certification payload Verify: drsleepytiger

	pbCertification := &corepb.Certification{
		CertificateHash: tx.CertificateHash,
		Issuer:          tx.Issuer.Bytes(),
		Certified:       tx.Certified.Bytes(),
		IssueTime:       tx.IssueTime,
		ExpirationTime:  tx.ExpirationTime,
		RevocationTime:  int64(-1),
	}
	certificationBytes, err := proto.Marshal(pbCertification)
	if err != nil {
		return err
	}

	// Add certification to certified's account state
	if err := certified.PutData(coreState.CertReceivedPrefix, tx.CertificateHash, certificationBytes); err != nil {
		return err
	}
	if err := b.State().PutAccount(certified); err != nil {
		return err
	}

	// Add certification to issuer's account state
	issuer, err = b.State().GetAccount(tx.Issuer)
	if err != nil {
		return err
	}
	if err := issuer.PutData(coreState.CertIssuedPrefix, tx.CertificateHash, certificationBytes); err != nil {
		return err
	}
	if err := b.State().PutAccount(issuer); err != nil {
		return err
	}

	return nil
}

// Bandwidth returns bandwidth.
func (tx *AddCertificationTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1500, uint64(tx.size))
}

// PointChange returns account's point change when applying this transaction.
func (tx *AddCertificationTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

// RecoverFrom returns from account's address.
func (tx *AddCertificationTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}

// RevokeCertificationPayload is payload type for RevokeCertificationTx
type RevokeCertificationPayload struct {
	CertificateHash []byte
}

// FromBytes converts bytes to payload.
func (payload *RevokeCertificationPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.RevokeCertificationPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.CertificateHash = payloadPb.Hash
	return nil
}

// ToBytes returns marshaled RevokeCertificationPayload
func (payload *RevokeCertificationPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.RevokeCertificationPayload{
		Hash: payload.CertificateHash,
	}
	return proto.Marshal(payloadPb)
}

// RevokeCertificationTx is a structure for revoking certification
type RevokeCertificationTx struct {
	*core.Transaction
	Revoker         common.Address
	CertificateHash []byte
	size            int
}

var _ core.ExecutableTx = &RevokeCertificationTx{}

// NewRevokeCertificationTx returns RevokeCertificationTx
func NewRevokeCertificationTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(RevokeCertificationPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}
	if !common.IsHash(byteutils.Bytes2Hex(payload.CertificateHash)) {
		return nil, ErrCertHashInvalid
	}

	return &RevokeCertificationTx{
		Transaction:     tx,
		Revoker:         tx.From(),
		CertificateHash: payload.CertificateHash,
		size:            size,
	}, nil
}

// Execute RevokeCertificationTx
func (tx *RevokeCertificationTx) Execute(b *core.Block) error {
	issuer, err := b.State().GetAccount(tx.Revoker)
	if err != nil {
		return err
	}
	certBytes, err := issuer.GetData(coreState.CertIssuedPrefix, tx.CertificateHash)
	if err != nil {
		return err
	}

	pbCert := new(corepb.Certification)
	err = proto.Unmarshal(certBytes, pbCert)
	if err != nil {
		return err
	}
	// verify transaction
	if !byteutils.Equal(pbCert.Issuer, tx.Revoker.Bytes()) {
		return ErrCertRevokerInvalid
	}
	if pbCert.RevocationTime > int64(-1) {
		return ErrCertAlreadyRevoked
	}
	if pbCert.ExpirationTime < b.State().Timestamp() {
		return ErrCertAlreadyExpired
	}

	pbCert.RevocationTime = b.State().Timestamp()
	newCertBytes, err := proto.Marshal(pbCert)
	if err != nil {
		return err
	}
	// change cert on issuer's cert issued List
	err = issuer.PutData(coreState.CertIssuedPrefix, tx.CertificateHash, newCertBytes)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(issuer)
	if err != nil {
		return err
	}
	// change cert on certified's cert received list
	certAddr, err := common.BytesToAddress(pbCert.Certified)
	if err != nil {
		return err
	}
	certified, err := b.State().GetAccount(certAddr)
	if err != nil {
		return err
	}
	err = certified.PutData(coreState.CertReceivedPrefix, tx.CertificateHash, newCertBytes)
	if err != nil {
		return err
	}
	return b.State().PutAccount(certified)
}

// Bandwidth returns bandwidth.
func (tx *RevokeCertificationTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1500, uint64(tx.size))
}

// PointChange returns account's point change when applying this transaction.
func (tx *RevokeCertificationTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

// RecoverFrom returns from account's address.
func (tx *RevokeCertificationTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}
