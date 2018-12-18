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

package net

import (
	"fmt"
	"sync"

	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/willf/bloom"
)

const (
	// according to https://krisives.github.io/bloom-calculator/
	// Count (n) = 100000, Error (p) = 0.001
	maxCountOfRecvMessageInBloomFiler = 1000000
	bloomFilterOfRecvMessageArgM      = 14377588
	bloomFilterOfRecvMessageArgK      = 10
)

// BloomFilter manages bloom filter
type BloomFilter struct {
	argM             uint
	argK             uint
	bloomFilter      *bloom.BloomFilter
	bloomFilterMutex sync.Mutex
	counts           int
	maxCounts        int
}

// NewBloomFilter returns BloomFilter
func NewBloomFilter(m uint, k uint, maxCount int) *BloomFilter {
	return &BloomFilter{
		argM:        m,
		argK:        k,
		bloomFilter: bloom.New(m, k),
		counts:      0,
		maxCounts:   maxCount,
	}
}

// RecordKey add key to bloom filter.
func (bf *BloomFilter) RecordKey(key string) {
	bf.bloomFilterMutex.Lock()
	defer bf.bloomFilterMutex.Unlock()

	bf.counts++
	if bf.counts > bf.maxCounts {
		// reset.
		logging.WithFields(logrus.Fields{
			"countOfBloomFilter": bf.counts,
		}).Debug("reset bloom filter.")
		bf.counts = 0
		bf.bloomFilter = bloom.New(bf.argM, bf.argK)
	}

	bf.bloomFilter.AddString(key)
}

// HasKey use bloom filter to check if the key exists quickly
func (bf *BloomFilter) HasKey(key string) bool {
	bf.bloomFilterMutex.Lock()
	defer bf.bloomFilterMutex.Unlock()

	return bf.bloomFilter.TestString(key)
}

// HasRecvMessage use bloom filter sender check if the key exists quickly
func (bf *BloomFilter) HasRecvMessage(msg *SendMessage) bool {
	bf.bloomFilterMutex.Lock()
	defer bf.bloomFilterMutex.Unlock()

	key := fmt.Sprintf("%s-%v", msg.receiver.Pretty(), msg.Hash())

	return bf.bloomFilter.TestString(key)
}

// RecordRecvMessage records received message
func (bf *BloomFilter) RecordRecvMessage(msg *RecvMessage) {
	key := fmt.Sprintf("%s-%v", msg.sender.Pretty(), msg.Hash())
	//fmt.Println("add bf:", key, "-", msg.MessageType())
	bf.RecordKey(key)
}
