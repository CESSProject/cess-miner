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

package node

import (
	"sync"

	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/pbc"
)

type Report struct {
	Cert      string
	Ias_sig   string
	Quote     string
	Quote_sig string
}

type Challenge struct {
	ChalID   []int          `json:"chal_id"`
	TimeOut  int            `json:"time_out"`
	QElement []pbc.QElement `json:"q_elements"`
}

type Status struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type ChalResponse struct {
	Challenge `json:"challenge"`
	Status    `json:"status"`
}

type ProofResult struct {
	Msg string `json:"message"`
}

type MinerProof struct {
	Sigma []byte   `json:"sigma"`
	Miu   [][]byte `json:"miu"`
	Tag   pbc.T    `json:"tag"`
}

type ChallengeResult struct {
	AutonomousBloomFilter      []int64 `json:"autonomous_bloom_filter"`
	IdleBloomFilter            []int64 `json:"idle_bloom_filter"`
	ServiceBloomFilter         []int64 `json:"service_bloom_filter"`
	AutonomousFailedFileHashes string  `json:"autonomous_failed_file_hashes"`
	IdleFailedFileHashes       string  `json:"idle_failed_file_hashes"`
	ServiceFailedFileHashes    string  `json:"service_failed_file_hashes"`
	ChalId                     []byte  `json:"chal_id"`
	Pkey                       []byte  `json:"pkey"`
	Sig                        []byte  `json:"sig"`
}

type ChallengeSignMessage struct {
	AutonomousBloomFilter      []int64 `json:"autonomous_bloom_filter"`
	IdleBloomFilter            []int64 `json:"idle_bloom_filter"`
	ServiceBloomFilter         []int64 `json:"service_bloom_filter"`
	AutonomousFailedFileHashes string  `json:"autonomous_failed_file_hashes"`
	IdleFailedFileHashes       string  `json:"idle_failed_file_hashes"`
	ServiceFailedFileHashes    string  `json:"service_failed_file_hashes"`
	ChalId                     string  `json:"chal_id"`
	Pkey                       string  `json:"pkey"`
}

const (
	Proof_Autonomous uint = 1
	Proof_Idle       uint = 2
	Proof_Service    uint = 3
)

const (
	M_Pending  = "pending"
	M_Positive = "positive"
	M_Frozen   = "frozen"
	M_Exit     = "exit"
)

const (
	Cach_Blockheight = "blockheight:"
	Chal_Blockheight = "challengeheight:"
)

var (
	Ch_Report      chan Report
	Ch_Tag         chan chain.Result
	Ch_ProofResult chan ChalResponse
	Ch_Challenge   chan ChalResponse
	chanllengeLock *sync.Mutex
)

func init() {
	Ch_Report = make(chan Report, 1)
	Ch_Tag = make(chan chain.Result, 1)
	Ch_ProofResult = make(chan ChalResponse, 1)
	Ch_Challenge = make(chan ChalResponse, 1)
	chanllengeLock = new(sync.Mutex)
}

func LockChallengeLock() {
	chanllengeLock.Lock()
}

func ReleaseChallengeLock() {
	chanllengeLock.Unlock()
}
