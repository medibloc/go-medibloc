// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package sync

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

//Download start sync download
func (s *Service) Download(bd *core.BlockData) error {
	s.mu.Lock()
	if s.downloading {
		return ErrDownloadActivated
	}
	s.downloading = true
	s.mu.Unlock()

	s.subscribeMap = new(sync.Map)
	s.numberOfRequests = 0
	s.targetHash = bd.Hash()
	s.targetHeight = bd.Height()
	s.downloadErrCh = make(chan error)

	logging.Console().WithFields(logrus.Fields{
		"targetHash":   bd.HexHash(),
		"targetHeight": bd.Height(),
	}).Info("Sync: Download is started.")
	go s.download()
	return nil
}

func (s *Service) download() {
	downloadCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	s.baseBlock, err = s.findBaseBlock()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warning("Sync: failed to find base block")
	} else if err := s.stepUpRequest(downloadCtx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warning("Sync: failed to step-up download")
	}

	logging.Console().WithFields(logrus.Fields{
		"targetHash":   byteutils.Bytes2Hex(s.targetHash),
		"targetHeight": s.targetHeight,
		"err":          err,
	}).Info("Sync: Download is stopped")

	s.mu.Lock()
	s.downloading = false
	s.mu.Unlock()
}

func (s *Service) findBaseBlock() (*core.BlockData, error) {
	base := s.bm.LIB().BlockData
	low := base.Height()
	high := s.targetHeight

	for {
		select {
		case <-s.ctx.Done():
			return nil, ErrContextDone
		default:
		}

		try := (high + low) / 2
		if try == low {
			break
		}
		query, err := s.newFindBaseRequest(try)
		if err != nil {
			return nil, err
		}

		responseCh := make(chan net.Message)
		s.netService.Register(net.NewSubscriber(query, responseCh, false, query.Id, net.MessageWeightZero))
		rf := &net.RandomPeerFilter{N: SimultaneousRequest}
		retry := 0
		for {
			select {
			case <-s.ctx.Done():
				return nil, ErrContextDone
			default:
			}
			retry++
			if retry > s.numberOfRetries {
				s.netService.Deregister(net.NewSubscriber(query, responseCh, false, query.Id, net.MessageWeightZero))
				return nil, ErrLimitedRetry
			}

			peers := s.netService.SendPbMessageToPeers(BaseSearch, query, net.MessagePriorityHigh, rf)
			if len(peers) == 0 {
				return nil, ErrFailedToConnect
			}
			success := false
			timeout := time.NewTimer(s.responseTimeLimit)
			for i := 0; i < len(peers); i++ {
				select {
				case <-s.ctx.Done():
					break
				case <-timeout.C:
					break
				case msg := <-responseCh:
					bd, err := s.handleFindBaseResponse(msg)
					if err != nil {
						continue
					}
					if bd == nil {
						high = try
					} else {
						low = try
						base = bd
					}
					success = true
					break
				}
			}
			if success {
				s.netService.Deregister(net.NewSubscriber(query, responseCh, false, query.Id, net.MessageWeightZero))
				break
			}
		}
	}
	return base, nil
}

func (s *Service) newFindBaseRequest(tryHeight uint64) (*syncpb.FindBaseRequest, error) {
	query := &syncpb.FindBaseRequest{
		TryHeight:    tryHeight,
		TargetHeight: s.targetHeight,
		Timestamp:    time.Now().Unix(),
	}
	id, err := hash.Sha3256Pb(query)
	if err != nil {
		return nil, err
	}
	// make query find mid height
	query.Id = "syncbase_" + byteutils.Bytes2Hex(id)
	return query, nil
}

func (s *Service) handleFindBaseRequest(msg net.Message) {
	req := new(syncpb.FindBaseRequest)
	if err := proto.Unmarshal(msg.Data(), req); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"sender": msg.MessageFrom(),
			"err":    err,
		}).Debug("failed to unmarshal msg")
		return //TODO: blacklist?
	}

	res := new(syncpb.FindBaseResponse)
	defer s.netService.SendPbMessageToPeer(req.Id, res, net.MessagePriorityLow, msg.MessageFrom())

	var err error
	res.TargetHash, err = s.bm.BlockHashByHeight(req.TargetHeight)
	if err != nil {
		res.Status = false
		return
	}
	res.TryHash, err = s.bm.BlockHashByHeight(req.TryHeight)
	if err != nil {
		res.Status = false
	}
	res.Status = true
}

func (s *Service) handleFindBaseResponse(msg net.Message) (*core.BlockData, error) {
	res := new(syncpb.FindBaseResponse)
	if err := proto.Unmarshal(msg.Data(), res); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"sender": msg.MessageFrom(),
			"err":    err,
		}).Debug("failed to unmarshal msg")
		return nil, err //TODO: blacklist?
	}
	if !res.Status {
		return nil, ErrNotFound //TODO: blacklist?
	}
	if !byteutils.Equal(s.targetHash, res.TargetHash) {
		return nil, ErrDifferentTargetHash
	}
	b := s.bm.BlockByHash(res.TryHash)
	if b == nil {
		return nil, nil
	}
	return b.BlockData, nil
}

func (s *Service) stepUpRequest(ctx context.Context) error {
	height := s.baseBlock.Height()
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-s.ctx.Done():
			return ErrContextDone
		case err := <-s.downloadErrCh:
			if err != nil {
				return err
			}
			s.numberOfRequests--
		case <-ticker.C:
			if s.numberOfRequests >= s.activeDownloadLimit {
				continue
			}
			s.numberOfRequests++
			height++
			if height > s.targetHeight {
				return nil
			}
			go s.downloadBlockByHeight(ctx, height)
		}
	}
}

func (s *Service) downloadBlockByHeight(ctx context.Context, height uint64) {
	query, err := s.newBlockByHeightRequest(height)
	if err != nil {
		s.downloadErrCh <- err
		return
	}
	s.subscribeMap.Store(query.Id, query.BlockHeight)
	defer s.subscribeMap.Delete(query.Id)

	responseCh := make(chan net.Message)

	s.netService.Register(net.NewSubscriber(query, responseCh, false, query.Id, net.MessageWeightZero))
	defer s.netService.Deregister(net.NewSubscriber(query, responseCh, false, query.Id, net.MessageWeightZero))

	rf := &net.RandomPeerFilter{N: SimultaneousRequest}
	retry := 0
	for {
		select {
		case <-s.ctx.Done():
			s.downloadErrCh <- ErrContextDone
		case <-ctx.Done():
			s.downloadErrCh <- ErrContextDone
		default:
		}

		retry++
		if retry > s.numberOfRetries {
			s.downloadErrCh <- ErrLimitedRetry
			return
		}
		peers := s.netService.SendPbMessageToPeers(BlockRequest, query, net.MessagePriorityHigh, rf)
		if len(peers) == 0 {
			s.downloadErrCh <- ErrFailedToConnect
			return
		}
		timeout := time.NewTimer(s.responseTimeLimit)
		for i := 0; i < len(peers); i++ {
			select {
			case <-s.ctx.Done():
			case <-ctx.Done():
			case <-timeout.C:
			case msg := <-responseCh:
				if err := s.handleBlockByHeightResponse(msg); err != nil {
					continue
				}
				s.downloadErrCh <- err
				return
			}
		}
	}
}

func (s *Service) newBlockByHeightRequest(height uint64) (*syncpb.BlockByHeightRequest, error) {
	query := &syncpb.BlockByHeightRequest{
		TargetHeight: s.targetHeight,
		BlockHeight:  height,
		Timestamp:    time.Now().Unix(),
	}
	id, err := hash.Sha3256Pb(query)
	if err != nil {
		return nil, err
	}
	query.Id = "syncblock_" + byteutils.Bytes2Hex(id)
	return query, nil
}

func (s *Service) handleBlockByHeightRequest(msg net.Message) {
	req := new(syncpb.BlockByHeightRequest)
	if err := proto.Unmarshal(msg.Data(), req); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"sender": msg.MessageFrom(),
			"err":    err,
		}).Debug("failed to unmarshal msg")
		return //TODO: blacklist?
	}

	res := new(syncpb.BlockByHeightResponse)
	defer s.netService.SendPbMessageToPeer(req.Id, res, net.MessagePriorityLow, msg.MessageFrom())

	var err error
	res.TargetHash, err = s.bm.BlockHashByHeight(req.TargetHeight)
	if err != nil {
		res.Status = false
		return
	}
	b, err := s.bm.BlockByHeight(req.BlockHeight)
	if err != nil {
		res.Status = false
		return
	}
	pb, err := b.ToProto()
	if err != nil {
		res.Status = false
		return
	}
	res.BlockData = pb.(*corepb.Block)
	res.Status = true
}

func (s *Service) handleBlockByHeightResponse(msg net.Message) error {
	res := new(syncpb.BlockByHeightResponse)
	if err := proto.Unmarshal(msg.Data(), res); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"sender": msg.MessageFrom(),
			"err":    err,
		}).Debug("failed to unmarshal msg")
		return err //TODO: blacklist?
	}
	if !res.Status {
		return ErrNotFound //TODO: blacklist?
	}
	if !byteutils.Equal(s.targetHash, res.TargetHash) {
		return ErrDifferentTargetHash
	}
	bd := new(core.BlockData)
	if err := bd.FromProto(res.BlockData); err != nil {
		return err
	}
	resHeight, ok := s.subscribeMap.Load(msg.MessageType())
	if !ok {
		return ErrCannotFindQueryID
	}
	if resHeight.(uint64) != bd.Height() {
		return ErrWrongHeightBlock
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	return s.bm.PushBlockDataSync2(ctx, bd)
}
