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

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	dataDir := "./data"

	cache, err := NewCache(dataDir, 0, 0, "scheduler")
	assert.NoError(t, err)

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

	//remove dir
	os.RemoveAll(dataDir)
}
