/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package cache

import "io"

type Reader interface {
	// Has returns true if the given key exists in the key-value data store.
	Has(key []byte) (bool, error)

	// Get fetch the given key if it's present in the key-value data store.
	Get(key []byte) ([]byte, error)

	// QueryPrefixKeyList queries the collection of all keys that start with
	// prefix but do not contain prefix
	QueryPrefixKeyList(prefix string) ([]string, error)
}

type Writer interface {
	// Put store the given key-value in the key-value data store
	Put(key []byte, value []byte) error

	// Delete removes the key from the key-value data store.
	Delete(key []byte) error
}

type Cache interface {
	Reader
	Writer
	io.Closer
}
