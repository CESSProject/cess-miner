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

package configs

// type and version
const Version = "cess-bucket v0.6.0 dev"

const (
	// Name is the name of the program
	Name = "cess-bucket"
	// Description is the description of the program
	Description = "The storage miner implementation of the CESS platform"
	// NameSpace is the cached namespace
	NameSpace = "bucket"
)

const (
	// BaseDir is the base directory where data is stored
	BaseDir = NameSpace
	// Data directory
	LogDir    = "log"
	CacheDir  = "cache"
	FileDir   = "file"
	FillerDir = "filler"
	TmpDir    = "tmp"
)
