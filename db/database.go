// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package db

import (
  "strconv"
  "strings"
  "sync"
  "time"

  "github.com/syndtr/goleveldb/leveldb"
  "github.com/syndtr/goleveldb/leveldb/errors"
  "github.com/syndtr/goleveldb/leveldb/filter"
  "github.com/syndtr/goleveldb/leveldb/iterator"
  "github.com/syndtr/goleveldb/leveldb/opt"
)

var OpenFileLimit = 64

type LDBDatabase struct {
  fn string      // filename for reporting
  db *leveldb.DB // LevelDB instance
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
  // Ensure we have some minimal caching and file guarantees
  if cache < 16 {
    cache = 16
  }
  if handles < 16 {
    handles = 16
  }

  // Open the db and recover any potential corruptions
  db, err := leveldb.OpenFile(file, &opt.Options{
    OpenFilesCacheCapacity: handles,
    BlockCacheCapacity:     cache / 2 * opt.MiB,
    WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
    Filter:                 filter.NewBloomFilter(10),
  })
  if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
    db, err = leveldb.RecoverFile(file, nil)
  }
  // (Re)check for errors and abort if opening of the db failed
  if err != nil {
    return nil, err
  }
  return &LDBDatabase{
    fn: file,
    db: db,
  }, nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
  return db.fn
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
  return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
  return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
  dat, err := db.db.Get(key, nil)
  if err != nil {
    return nil, err
  }
  return dat, nil
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
  return db.db.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
  return db.db.NewIterator(nil, nil)
}

func (db *LDBDatabase) Close() {
  err := db.db.Close()
  if err == nil {
    // db.log.Info("Database closed")
  } else {
    // db.log.Error("Failed to close database", "err", err)
  }
}

func (db *LDBDatabase) LDB() *leveldb.DB {
  return db.db
}

func (db *LDBDatabase) NewBatch() Batch {
  return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type ldbBatch struct {
  db   *leveldb.DB
  b    *leveldb.Batch
  size int
}

func (b *ldbBatch) Put(key, value []byte) error {
  b.b.Put(key, value)
  b.size += len(value)
  return nil
}

func (b *ldbBatch) Write() error {
  return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
  return b.size
}

func (b *ldbBatch) Reset() {
  b.b.Reset()
  b.size = 0
}

type table struct {
  db     Database
  prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
  return &table{
    db:     db,
    prefix: prefix,
  }
}

func (dt *table) Put(key []byte, value []byte) error {
  return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
  return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
  return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
  return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Close() {
  // Do nothing; don't close the underlying DB.
}

type tableBatch struct {
  batch  Batch
  prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
  return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
  return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
  return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
  return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
  return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
  tb.batch.Reset()
}
