package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	dpospb "github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
)

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

// VoteTx is a structure for voting
type VoteTx struct {
	voter        common.Address
	candidateIDs [][]byte
	size         int
}

var _ core.ExecutableTx = &VoteTx{}

// NewVoteTx returns VoteTx
func NewVoteTx(tx *coreState.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(VotePayload)
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

	return &VoteTx{
		voter:        tx.From(),
		candidateIDs: payload.CandidateIDs,
		size:         size,
	}, nil
}

// Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	if len(tx.candidateIDs) > VoteMaximum {
		return ErrOverMaxVote
	}

	acc, err := b.State().GetAccount(tx.voter)
	if err != nil {
		return err
	}

	newVoted, err := trie.NewBatch(nil, acc.Storage)
	if err != nil {
		return err
	}
	err = newVoted.Prepare()
	if err != nil {
		return err
	}
	for _, c := range tx.candidateIDs {
		_, err := newVoted.Get(c)
		// Check duplicated vote
		if err != nil && err != trie.ErrNotFound {
			return err
		} else if err == nil {
			return ErrDuplicateVote
		}

		// Add candidate to voters voted
		if err := newVoted.Put(c, c); err != nil {
			return err
		}
	}
	err = newVoted.Flush()
	if err != nil {
		return err
	}

	ds := b.State().DposState()

	// Add vote power to new voted
	iter, err := newVoted.Iterator(nil)
	if err != nil {
		return err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return err
		}
		if !exist {
			break
		}

		candidateID := iter.Key()
		err = ds.AddVotePowerToCandidate(candidateID, acc.Staking)
		if err == trie.ErrNotFound {
			return ErrNotCandidate
		} else if err != nil {
			return err
		}
	}

	oldVoted := acc.Voted
	// Subtract vote power from old voted
	iter, err = oldVoted.Iterator(nil)
	if err != nil {
		return err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return err
		}
		if !exist {
			break
		}

		candidateID := iter.Key()
		err = ds.SubVotePowerToCandidate(candidateID, acc.Staking)
		if err == trie.ErrNotFound {
			continue // candidate quited
		} else if err != nil {
			return err
		}
	}

	// change voter's account
	acc.Voted = newVoted
	if err := b.State().PutAccount(acc); err != nil {
		return err
	}

	return nil
}

// Bandwidth returns bandwidth.
func (tx *VoteTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

func (tx *VoteTx) PointModifier(points *util.Uint128) (modifiedPoints *util.Uint128, err error) {
	return points, nil
}
