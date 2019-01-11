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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
	syncpb "github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type download struct {
	ctx     context.Context
	cancel  context.CancelFunc
	service *Service

	bm           BlockManager
	targetHeight uint64
	targetHash   []byte

	try  uint64
	low  uint64
	high uint64

	baseBlock *core.BlockData
}

//Download start sync download
func (s *Service) Download(bd *core.BlockData) error {
	if s.IsDownloadActivated() {
		return ErrDownloadActivated
	}
	s.mu.Lock()
	s.downloading = true
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	d := &download{
		ctx:          ctx,
		cancel:       cancel,
		service:      s,
		bm:           s.bm,
		targetHeight: bd.Height(),
		targetHash:   bd.Hash(),

		baseBlock: nil,
	}

	logging.Console().WithFields(logrus.Fields{
		"targetHash":   bd.HexHash(),
		"targetHeight": bd.Height(),
	}).Info("Sync: Download is started.")

	go d.start()

	go func() {
		select {
		case <-s.ctx.Done():
			d.stop()
		}
	}()

	return nil
}

func (d *download) start() {
	defer d.stop()
	if err := d.findBaseBlock(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warning("Sync: failed to find base block")
		return
	}

	if err := d.stepUpRequest(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warning("Sync: failed to step-up download")
		return
	}

	logging.Console().WithFields(logrus.Fields{
		"targetHash":   byteutils.Bytes2Hex(d.targetHash),
		"targetHeight": d.targetHeight,
	}).Info("Sync: Download is finished")

}

func (d *download) stop() {
	d.cancel()

	d.service.mu.Lock()
	d.service.downloading = false
	d.service.mu.Unlock()
}

func (d *download) findBaseBlock() error {
	ss := d.service
	d.baseBlock = d.bm.LIB().BlockData
	d.low = d.baseBlock.Height()
	d.high = d.targetHeight

	for {
		select {
		case <-d.ctx.Done():
			return ErrDownloadCtxDone
		default:
		}

		d.try = (d.high + d.low) / 2
		if d.try == d.low {
			break
		}
		query := newFindBaseRequest(d)
		rf := &net.RandomPeerFilter{N: SimultaneousRequest}

		var ok bool
		for retry := 0; retry < ss.numberOfRetries; retry++ {
			ok = func() bool {
				ctx, cancel := context.WithTimeout(d.ctx, ss.responseTimeLimit)
				defer cancel()
				ok, errs := d.service.netService.RequestAndResponse(ctx, query, d.handleFindBaseResponse, rf)
				for _, err := range errs {
					if err != nil {
						logging.Console().WithFields(logrus.Fields{
							"err": err,
						}).Warning("error occurred during find base block request")
					}
				}
				return ok
			}()
			if ok {
				break
			}
		}
		if !ok {
			return ErrLimitedRetry
		}
	}
	return nil
}

func (d *download) handleFindBaseResponse(_ net.Query, msg net.Message) error {
	res := new(syncpb.FindBaseResponse)
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
	if !byteutils.Equal(d.targetHash, res.TargetHash) {
		return ErrDifferentTargetHash
	}
	b := d.bm.BlockByHash(res.TryHash)
	if b == nil {
		d.high = d.try
	} else {
		d.low = d.try
		d.baseBlock = b.BlockData
	}
	return nil
}

func (d *download) stepUpRequest() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error)

	numberOfRequests := 0

	height := d.baseBlock.Height()
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-d.ctx.Done():
			return ErrContextDone
		case err := <-errCh:
			if err != nil {
				return err
			}
			numberOfRequests--
		case <-ticker.C:
			if numberOfRequests >= d.service.activeDownloadLimit {
				continue
			}
			numberOfRequests++
			height++
			if height > d.targetHeight {
				return nil
			}
			go d.downloadBlockByHeight(ctx, errCh, height)
		}
	}
}

func (d *download) downloadBlockByHeight(ctx context.Context, errCh chan<- error, height uint64) {
	ss := d.service

	rf := &net.RandomPeerFilter{N: SimultaneousRequest}
	query := newBlockByHeightRequest(d, height)

	var ok bool
	for retry := 0; retry < ss.numberOfRetries; retry++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ok = func() bool {
			ctx, cancel := context.WithTimeout(ctx, ss.responseTimeLimit)
			defer cancel()
			ok, errs := d.service.netService.RequestAndResponse(ctx, query, d.handleBlockByHeightResponse, rf)
			for _, err := range errs {
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"height": height,
						"err":    err,
					}).Warning("error occurred during download Block by Height request")
				}
			}
			return ok
		}()
		if ok {
			break
		}
	}
	if !ok {
		errCh <- ErrLimitedRetry
	}
	errCh <- nil
}

func (d *download) handleBlockByHeightResponse(q net.Query, msg net.Message) error {
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
	if !byteutils.Equal(d.targetHash, res.TargetHash) {
		return ErrDifferentTargetHash
	}
	bd := new(core.BlockData)
	if err := bd.FromProto(res.BlockData); err != nil {
		return err
	}

	req := q.(*DownloadByHeightQuery)
	if req.pb.BlockHeight != bd.Height() {
		return ErrWrongHeightBlock
	}
	ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
	defer cancel()

	return d.bm.PushBlockDataSync2(ctx, bd)
}
