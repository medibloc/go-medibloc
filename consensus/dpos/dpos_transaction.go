package dpos

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// BecomeCandidateTx is a structure for quiting cadidate
type BecomeCandidateTx struct {
	candidateAddr common.Address
	collateral    *util.Uint128
}


//NewBecomeCandidateTx returns BecomeCandidateTx
func NewBecomeCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &BecomeCandidateTx{
		candidateAddr: tx.From(),
		collateral:    tx.Value(),
	}, nil
}

//Execute NewBecomeCandidateTx
func (tx *BecomeCandidateTx) Execute(b *core.Block) error {
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	_, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrAlreadyInCandidacy
	}

	err = as.SubBalance(tx.candidateAddr.Bytes(), tx.collateral)
	if err != nil {
		return err
	}

	collateralBytes, err := tx.collateral.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}

	pbCandidate := &dpospb.Candidate{
		Address:   tx.candidateAddr.Bytes(),
		Collatral: collateralBytes,
		VotePower: make([]byte, 0),
	}
	candidateBytes, err := proto.Marshal(pbCandidate)
	if err != nil {
		return err
	}

	return cs.Put(tx.candidateAddr.Bytes(), candidateBytes)
}

//QuitCandidateTx is a structure for quiting cadidate
type QuitCandidateTx struct {
	candidateAddr common.Address
}

//NewQuitCandidateTx returns QuitCandidateTx
func NewQuitCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &QuitCandidateTx{
		candidateAddr: tx.From(),
	}, nil
}

//Execute QuitCandidateTx
func (tx *QuitCandidateTx) Execute(b *core.Block) error {
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	candidateBytes, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
		return err
	}

	// Refund collateral
	pbCandidate := new(dpospb.Candidate)
	err = proto.Unmarshal(candidateBytes, pbCandidate)
	if err != nil {
		return err
	}
	collateral, err := util.NewUint128FromFixedSizeByteSlice(pbCandidate.Collatral)
	if err != nil {
		return err
	}
	if err := as.AddBalance(tx.candidateAddr.Bytes(), collateral); err != nil {
		return err
	}

	return cs.Delete(tx.candidateAddr.Bytes())
}

//VoteTx is a structure for voting
type VoteTx struct {
	voter         common.Address
	candidateAddr common.Address
}

//NewVoteTx returns VoteTx
func NewVoteTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &VoteTx{
		voter:         tx.From(),
		candidateAddr: tx.To(),
	}, nil
}

//Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	candidateBytes, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
		return err
	}

	voter, err := as.AccountState().GetAccount(tx.voter.Bytes())
	if err != nil {
		return err
	}

	// Set voter's voted candidate
	if byteutils.Equal(voter.Voted(), tx.candidateAddr.Bytes()) {
		return ErrVoteDuplicate
	}
	as.SetVoted(tx.voter.Bytes(), tx.candidateAddr.Bytes())

	// Add voter's vesting to candidate's votePower
	pbCandidate := new(dpospb.Candidate)
	err = proto.Unmarshal(candidateBytes, pbCandidate)
	if err != nil {
		return err
	}
	votePower, err := util.NewUint128FromFixedSizeByteSlice(pbCandidate.VotePower)
	if err != nil {
		return err
	}
	newVotePower, err := votePower.Add(voter.Vesting())
	if err != nil {
		return err
	}
	pbCandidate.VotePower, err = newVotePower.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}
	newBytesCandidate, err := proto.Marshal(pbCandidate)
	if err != nil {
		return err
	}

	return cs.Put(tx.candidateAddr.Bytes(), newBytesCandidate)
}
