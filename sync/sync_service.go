package sync

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
)

type Service struct {
	blockChain *core.BlockChain
	netService net.Service

	quitCh    chan bool
	messageCh chan net.Message
}

func NewService() *Service {
	return &Service{

	}
}

func (ss *Service) start(){

}

func (ss *Service) stop(){

}

