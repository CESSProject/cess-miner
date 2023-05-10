/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	// minCache is the minimum amount of memory in megabytes
	// to allocate to leveldb.
	minCache = 16

	// minHandles is the minimum number of files handles to
	// allocate to the open database files.
	minHandles = 32
)

type LevelDB struct {
	fn string
	db *leveldb.DB
	l  *sync.RWMutex
}

var (
	NotFound = leveldb.ErrNotFound
)

func NewCache(fpath string, memory int, handles int, namespace string) (Cache, error) {
	_, err := os.Stat(fpath)
	if err != nil {
		err = os.MkdirAll(fpath, configs.DirMode)
		if err != nil {
			return nil, err
		}
	}
	return newLevelDB(fpath, memory, handles, namespace)
}

func newLevelDB(file string, memory int, handles int, namespace string) (Cache, error) {
	options := configureOptions(memory, handles)
	db, err := leveldb.OpenFile(file, options)
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	if err != nil {
		return nil, err
	}

	ldb := &LevelDB{
		fn: file,
		db: db,
		l:  new(sync.RWMutex),
	}
	return ldb, nil
}

func configureOptions(cache int, handles int) *opt.Options {
	// Set default options
	options := &opt.Options{
		Filter:                 filter.NewBloomFilter(10),
		DisableSeeksCompaction: true,
	}
	if cache < minCache {
		cache = minCache
	}
	if handles < minHandles {
		handles = minHandles
	}
	// Set default options
	options.OpenFilesCacheCapacity = handles
	options.BlockCacheCapacity = cache / 2 * opt.MiB
	options.WriteBuffer = cache / 4 * opt.MiB

	return options
}

func (db *LevelDB) Close() error {
	db.l.Lock()
	defer db.l.Unlock()
	return db.db.Close()
}

func (db *LevelDB) Has(key []byte) (bool, error) {
	db.l.RLock()
	defer db.l.RUnlock()
	return db.db.Has(key, nil)
}

func (db *LevelDB) Get(key []byte) ([]byte, error) {
	db.l.RLock()
	defer db.l.RUnlock()
	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LevelDB) Put(key []byte, value []byte) error {
	db.l.Lock()
	defer db.l.Unlock()
	return db.db.Put(key, value, nil)
}

func (db *LevelDB) Delete(key []byte) error {
	db.l.Lock()
	defer db.l.Unlock()
	return db.db.Delete(key, nil)
}

func (db *LevelDB) Compact(start []byte, limit []byte) error {
	db.l.RLock()
	defer db.l.RUnlock()
	return db.db.CompactRange(util.Range{Start: start, Limit: limit})
}

func (db *LevelDB) QueryPrefixKeyList(prefix string) ([]string, error) {
	var result = make([]string, 0)
	db.l.RLock()
	defer db.l.RUnlock()
	iter := db.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		result = append(result, strings.TrimPrefix(string(iter.Key()), prefix))
	}
	iter.Release()
	return result, iter.Error()
}

func (db *LevelDB) QueryPrefixKeyListByHeigh(prefix string, blockheight uint32) ([]string, error) {
	var result = make([]string, 0)
	var height uint64
	var err error
	db.l.RLock()
	defer db.l.RUnlock()
	iter := db.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		height, err = strconv.ParseUint(string(iter.Value()), 10, 32)
		if err != nil {
			continue
		}
		if blockheight >= uint32(height) {
			result = append(result, strings.TrimPrefix(string(iter.Key()), prefix))
		}
	}
	iter.Release()
	return result, iter.Error()
}
