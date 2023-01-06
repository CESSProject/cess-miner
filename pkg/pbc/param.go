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

package pbc

// T be the file tag for F
type T struct {
	Tag   `json:"t"`
	MacT0 []byte `json:"mac_t0"`
}

// Tag belongs to T
type Tag struct {
	N        int64  `json:"n"`
	Enc      []byte `json:"enc"`
	FileHash []byte `json:"file_hash"`
}

type StatusInfo struct {
	StatusCode uint   `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type QElement struct {
	I int64 `json:"i"`
	V int64 `json:"v"`
}
