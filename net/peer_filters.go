package net

import (
	"math/rand"
)

// ChainSyncPeersFilter will filter some peers randomly
type ChainSyncPeersFilter struct {
}

// Filter implements PeerFilterAlgorithm interface
func (filter *ChainSyncPeersFilter) Filter(peers PeersSlice) PeersSlice {
	return peers
}

// RandomPeerFilter will filter a peer randomly
type RandomPeerFilter struct {
}

// Filter implements PeerFilterAlgorithm interface
func (filter *RandomPeerFilter) Filter(peers PeersSlice) PeersSlice {
	if len(peers) == 0 {
		return peers
	}

	selection := rand.Intn(len(peers))
	return peers[selection : selection+1]
}
