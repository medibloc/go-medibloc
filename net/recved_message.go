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

var (
	bloomFilterOfRecvMessage        = bloom.New(bloomFilterOfRecvMessageArgM, bloomFilterOfRecvMessageArgK)
	bloomFilterMutex                sync.Mutex
	countOfRecvMessageInBloomFilter = 0
)

// RecordKey add key to bloom filter.
func RecordKey(key string) {
	bloomFilterMutex.Lock()
	defer bloomFilterMutex.Unlock()

	countOfRecvMessageInBloomFilter++
	if countOfRecvMessageInBloomFilter > maxCountOfRecvMessageInBloomFiler {
		// reset.
		logging.VLog().WithFields(logrus.Fields{
			"countOfRecvMessageInBloomFilter": countOfRecvMessageInBloomFilter,
		}).Debug("reset bloom filter.")
		countOfRecvMessageInBloomFilter = 0
		bloomFilterOfRecvMessage = bloom.New(bloomFilterOfRecvMessageArgM, bloomFilterOfRecvMessageArgK)
	}

	bloomFilterOfRecvMessage.AddString(key)
}

// HasKey use bloom filter to check if the key exists quickly
func HasKey(key string) bool {
	bloomFilterMutex.Lock()
	defer bloomFilterMutex.Unlock()

	return bloomFilterOfRecvMessage.TestString(key)
}

// RecordRecvMessage records received message
func RecordRecvMessage(s *Stream, hash uint32) {
	RecordKey(fmt.Sprintf("%s-%d", s.pid, hash))
}

// HasRecvMessage check if the received message exists before
func HasRecvMessage(s *Stream, hash uint32) bool {
	return HasKey(fmt.Sprintf("%s-%d", s.pid, hash))
}
