package core

import (
	"sort"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/util"
)

type candidate struct {
	address    common.Address
	votesPower *util.Uint128
	candidacy  bool
}

type candidates []*candidate

type votesCache struct {
	batching   bool
	candidates candidates
}

func newVotesCache() *votesCache {
	return &votesCache{
		candidates: candidates{},
	}
}

func (cs candidates) Len() int {
	return len(cs)
}

func (cs candidates) Less(i, j int) bool {
	return cs[i].votesPower.Cmp(cs[j].votesPower) < 0
}

func (cs candidates) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func (vc *votesCache) Clone() *votesCache {
	candidates := make(candidates, vc.candidates.Len())
	for i := 0; i < vc.candidates.Len(); i++ {
		candidates[i] = &candidate{
			address:    vc.candidates[i].address,
			votesPower: vc.candidates[i].votesPower.DeepCopy(),
		}
	}
	return &votesCache{
		candidates: candidates,
	}
}

func (vc *votesCache) GetCandidate(address common.Address) (int, *candidate, error) {
	var idx int
	for idx = 0; idx < vc.candidates.Len(); idx++ {
		if vc.candidates[idx].address == address {
			break
		}
	}
	if idx == vc.candidates.Len() {
		return -1, nil, ErrCandidateNotFound
	}
	return idx, vc.candidates[idx], nil
}

func (vc *votesCache) AddCandidate(address common.Address, votesPower *util.Uint128) {
	_, c, err := vc.GetCandidate(address)
	if err == nil {
		c.candidacy = true
		return
	}
	c = &candidate{
		address:    address,
		votesPower: votesPower,
		candidacy:  true,
	}
	vc.candidates = append(vc.candidates, c)
	sort.Sort(vc.candidates)
}

func (vc *votesCache) RemoveCandidate(address common.Address) error {
	idx, _, err := vc.GetCandidate(address)
	if err != nil {
		return nil
	}
	vc.candidates = append(vc.candidates[:idx], vc.candidates[idx+1:]...)
	return nil
}

func (vc *votesCache) AddVotesPower(address common.Address, amount *util.Uint128) error {
	_, c, err := vc.GetCandidate(address)
	if err != nil && err != ErrCandidateNotFound {
		return err
	}
	if err == ErrCandidateNotFound {
		vc.AddCandidate(address, amount)
		return nil
	}
	c.votesPower, err = c.votesPower.Add(amount)
	if err != nil {
		return err
	}
	sort.Sort(vc.candidates)
	return nil
}

func (vc *votesCache) SubtractVotesPower(address common.Address, amount *util.Uint128) error {
	_, c, err := vc.GetCandidate(address)
	if err != nil {
		return err
	}
	if c.votesPower.Cmp(amount) < 0 {
		return ErrVotesPowerGetsMinus
	}
	c.votesPower, err = c.votesPower.Sub(amount)
	if err != nil {
		return err
	}
	sort.Sort(vc.candidates)
	return nil
}

func (vc *votesCache) SetCandidacy(address common.Address, candidacy bool) error {
	_, c, err := vc.GetCandidate(address)
	if err != nil {
		return err
	}
	c.candidacy = candidacy
	return nil
}
