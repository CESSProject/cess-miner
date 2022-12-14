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

import "io"

type Reader interface {
	// Has returns true if the given key exists in the key-value data store.
	Has(key []byte) (bool, error)

	// Get fetch the given key if it's present in the key-value data store.
	Get(key []byte) ([]byte, error)
}

type Writer interface {
	// Put store the given key-value in the key-value data store
	Put(key []byte, value []byte) error

	// Delete removes the key from the key-value data store.
	Delete(key []byte) error
}

type ICache interface {
	Reader
	Writer
	io.Closer
}
