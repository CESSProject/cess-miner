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

package serve

import (
	"sync"
)

var pack_once sync.Once

type pack_factory struct{}

var factoryInstance *pack_factory

// Generate different packet unpacking methods, single example
func Factory() *pack_factory {
	pack_once.Do(func() {
		factoryInstance = new(pack_factory)
	})

	return factoryInstance
}

// NewPack creates a specific unpacking object
func (f *pack_factory) NewPack(kind string) IDataPack {
	var dataPack IDataPack

	switch kind {
	// Default packaging and unpacking methods
	case DefaultDataPack:
		dataPack = NewDataPack()
		break

		//case custom package method:

	default:
		dataPack = NewDataPack()
	}

	return dataPack
}
