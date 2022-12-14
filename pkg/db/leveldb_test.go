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
