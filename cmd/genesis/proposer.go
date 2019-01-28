package main

import (
	"github.com/gogo/protobuf/proto"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
)

func ProposerOutput(cont Container) []byte {
	proposers := make([]*medletpb.ProposerConfig, 0, len(cont.Secrets))
	for i, s := range cont.Secrets {
		if i >= int(cont.Config.DynastySize) {
			break
		}
		proposers = append(proposers, &medletpb.ProposerConfig{
			Proposer: s.Public,
			Privkey:  s.Private,
			Coinbase: s.Public,
		})
	}
	conf := &medletpb.Config{
		Chain: &medletpb.ChainConfig{
			Proposers: proposers,
		},
	}
	return []byte(proto.MarshalTextString(conf))
}
