package medlet

import (
	"github.com/medibloc/go-medibloc/medlet/pb"
)

// Medlet manages blockchain services.
type Medlet struct {
	config *medletpb.Config
}

// New returns a new medlet.
func New(config *medletpb.Config) (*Medlet, error) {
	return &Medlet{
		config: config,
	}, nil
}

// Config returns medlet configuration.
func (m *Medlet) Config() *medletpb.Config {
	return m.config
}
