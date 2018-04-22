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

// RecordRecvMessage records received message
func (bf *BloomFilter) RecordRecvMessage(s *Stream, hash uint32) {
	bf.RecordKey(fmt.Sprintf("%s-%d", s.pid.Pretty(), hash))
}

// HasRecvMessage check if the received message exists before
func (bf *BloomFilter) HasRecvMessage(s *Stream, hash uint32) bool {
	return bf.HasKey(fmt.Sprintf("%s-%d", s.pid.Pretty(), hash))
}
