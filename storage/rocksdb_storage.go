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

package storage

import (
	"sync"

	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tecbot/gorocksdb"
)

type batchOpt struct {
	key     []byte
	value   []byte
	deleted bool
}

// RocksStorage the nodes in trie.
type RocksStorage struct {
	db          *gorocksdb.DB
	enableBatch bool
	mutex       sync.Mutex
	batchOpts   map[string]*batchOpt

	ro *gorocksdb.ReadOptions
	wo *gorocksdb.WriteOptions

	cache *gorocksdb.Cache
}

// NewRocksStorage init a storage
func NewRocksStorage(path string) (*RocksStorage, error) {

	filter := gorocksdb.NewBloomFilter(10)
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(filter)

	cache := gorocksdb.NewLRUCache(512 << 20)
	bbto.SetBlockCache(cache)
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	opts.SetMaxOpenFiles(500)
	opts.SetWriteBufferSize(64 * opt.MiB) // Default: 4MB
	opts.IncreaseParallelism(4)           // flush and compaction thread

	db, err := gorocksdb.OpenDb(opts, path)
	if err != nil {
		return nil, err
	}

	storage := &RocksStorage{
		db:          db,
		cache:       cache,
		enableBatch: false,
		batchOpts:   make(map[string]*batchOpt),
		ro:          gorocksdb.NewDefaultReadOptions(),
		wo:          gorocksdb.NewDefaultWriteOptions(),
	}

	// go RecordMetrics(storage)

	return storage, nil
}

// Get return value to the key in Storage
func (storage *RocksStorage) Get(key []byte) ([]byte, error) {
	if storage.enableBatch {
		storage.mutex.Lock()
		defer storage.mutex.Unlock()

		Opts := storage.batchOpts[byteutils.Bytes2Hex(key)]
		if Opts != nil {
			if Opts.deleted {
				return nil, ErrKeyNotFound
			}
			return Opts.value, nil
		}
	}

	value, err := storage.db.GetBytes(storage.ro, key)

	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, ErrKeyNotFound
	}

	return value, err
}

// Put put the key-value entry to Storage
func (storage *RocksStorage) Put(key []byte, value []byte) error {
	if storage.enableBatch {
		storage.mutex.Lock()
		defer storage.mutex.Unlock()

		storage.batchOpts[byteutils.Bytes2Hex(key)] = &batchOpt{
			key:     key,
			value:   value,
			deleted: false,
		}

		return nil
	}

	return storage.db.Put(storage.wo, key, value)
}

// Delete delete the key in Storage.
func (storage *RocksStorage) Delete(key []byte) error {
	if storage.enableBatch {
		storage.mutex.Lock()
		defer storage.mutex.Unlock()

		storage.batchOpts[byteutils.Bytes2Hex(key)] = &batchOpt{
			key:     key,
			deleted: true,
		}

		return nil
	}
	return storage.db.Delete(storage.wo, key)
}

// Close levelDB
func (storage *RocksStorage) Close() error {
	storage.db.Close()
	return nil
}

// EnableBatch enable batch write.
func (storage *RocksStorage) EnableBatch() {
	return
	//	storage.enableBatch = true
}

// Flush write and flush pending batch write.
func (storage *RocksStorage) Flush() error {
	return nil
	// storage.mutex.Lock()
	// defer storage.mutex.Unlock()
	//
	// if !storage.enableBatch {
	//	return nil
	// }
	//
	// //startAt := time.Now().UnixNano()
	//
	// wb := gorocksdb.NewWriteBatch()
	// defer wb.Destroy()
	//
	// //bl := len(storage.batchOpts)
	//
	// for _, opt := range storage.batchOpts {
	//	if opt.deleted {
	//		wb.Delete(opt.key)
	//	} else {
	//		wb.Put(opt.key, opt.value)
	//	}
	// }
	// storage.batchOpts = make(map[string]*batchOpt)
	//
	// err := storage.db.Write(storage.wo, wb)
	//
	// //endAt := time.Now().UnixNano()
	// //metricsRocksdbFlushTime.Update(endAt - startAt)
	// //metricsRocksdbFlushLen.Update(int64(bl))
	//
	// return err
}

// DisableBatch disable batch write.
func (storage *RocksStorage) DisableBatch() {
	return
	// storage.mutex.Lock()
	// defer storage.mutex.Unlock()
	// storage.batchOpts = make(map[string]*batchOpt)
	//
	// storage.enableBatch = false
}

// RecordMetrics record rocksdb metrics
// func RecordMetrics(storage *RocksStorage) {
//	metricsUpdateChan := time.NewTicker(5 * time.Second).C
//
//	for {
//		select {
//		case <-metricsUpdateChan:
//
//			readersMemStr := storage.db.GetProperty("rocksdb.estimate-table-readers-mem")
//			allMemTablesStr := storage.db.GetProperty("rocksdb.cur-size-all-mem-tables")
//			//cacheSize := storage.cache.GetUsage()
//			//pinnedSize := storage.cache.GetPinnedUsage()
//
//			readersMem, err := strconv.Atoi(readersMemStr)
//			if err != nil {
//				break
//			}
//			allMemTables, err := strconv.Atoi(allMemTablesStr)
//			if err != nil {
//				break
//			}
//
//			//metricsBlocksdbAllMemTables.Update(int64(allMemTables))
//			//metricsBlocksdbTableReaderMem.Update(int64(readersMem))
//			//metricsBlocksdbCacheSize.Update(int64(cacheSize))
//			//metricsBlocksdbCachePinnedSize.Update(int64(pinnedSize))
//		}
//	}
// }
