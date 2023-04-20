/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	dataDir := fmt.Sprintf("./%v", time.Now().Nanosecond())

	cache, err := NewCache(dataDir, 0, 0, "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dataDir)

	// get nil
	_, err = cache.Get([]byte("nil"))
	if err.Error() != "leveldb: not found" {
		assert.NoError(t, err)
	}

	//put
	err = cache.Put([]byte("key1"), nil)
	assert.NoError(t, err)

	//has
	ok, err := cache.Has([]byte("key1"))
	assert.NoError(t, err)
	if !ok {
		assert.NoError(t, fmt.Errorf("cache.Has err"))
	}

	// get
	_, err = cache.Get([]byte("key1"))
	assert.NoError(t, err)

	// delete
	err = cache.Delete([]byte("key1"))
	assert.NoError(t, err)

	//has
	ok, err = cache.Has([]byte("key1"))
	assert.NoError(t, err)
	if ok {
		assert.NoError(t, fmt.Errorf("cache.Has err"))
	}

	const prefix = "prefix:"
	var keys = []string{"1", "2", "3"}
	for _, v := range keys {
		err = cache.Put([]byte(prefix+v), nil)
		assert.NoError(t, err)
	}
	err = cache.Put([]byte("1"), nil)
	assert.NoError(t, err)
	err = cache.Put([]byte("z"), nil)
	assert.NoError(t, err)
	err = cache.Put([]byte("prefix"), nil)
	assert.NoError(t, err)
	list, err := cache.QueryPrefixKeyList(prefix)
	assert.NoError(t, err)

	assert.Equal(t, keys, list)
}
