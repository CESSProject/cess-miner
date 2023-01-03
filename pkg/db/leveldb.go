/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package db

import (
	"os"

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

var (
	NotFound = leveldb.ErrNotFound
)

type LevelDB struct {
	fn string
	db *leveldb.DB
}

func NewCache(fpath string, memory int, handles int, namespace string) (ICache, error) {
	_, err := os.Stat(fpath)
	if err != nil {
		err = os.MkdirAll(fpath, os.ModeDir)
		if err != nil {
			return nil, err
		}
	}
	return newLevelDB(fpath, memory, handles, namespace)
}

func newLevelDB(file string, memory int, handles int, namespace string) (ICache, error) {
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
	return db.db.Close()
}

func (db *LevelDB) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

func (db *LevelDB) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LevelDB) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *LevelDB) Compact(start []byte, limit []byte) error {
	return db.db.CompactRange(util.Range{Start: start, Limit: limit})
}
