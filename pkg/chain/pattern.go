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

package chain

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

const (
	ERR_Failed  = "failed"
	ERR_Timeout = "timeout"
	ERR_Empty   = "empty"
)

// error type
var (
	ERR_RPC_CONNECTION = errors.New("rpc connection failed")
)

type FileHash [64]types.U8
type SliceId [68]types.U8
type Random [20]types.U8
type Signature [65]types.U8
type Filter [256]types.U64
type Public [33]types.U8

// storage miner info
type MinerInfo struct {
	Beneficiary    types.AccountID
	Ip             Ipv4Type
	Collaterals    types.U128
	Debt           types.U128
	State          types.Bytes
	Idle_space     types.U128
	Service_space  types.U128
	Autonomy_space types.U128
	Puk            Public
	Ias_cert       types.Bytes
	Bloom_filter   BloomCollect
}

type BloomCollect struct {
	AutonomyFilter Filter
	ServiceFilter  Filter
	IdleFilter     Filter
}

// file meta info
type FileMetaInfo struct {
	Size       types.U64
	Index      types.U32
	State      types.Bytes
	UserBriefs []UserBrief
	Blockups   []Backup
}

type UserBrief struct {
	User        types.AccountID
	File_name   types.Bytes
	Bucket_name types.Bytes
}

// Backups
type Backup struct {
	Backup_index types.U8
	State        types.Bool
	Slice_info   []SliceInfo
}

// SliceInfo
type SliceInfo struct {
	Shard_id   SliceId
	Slice_hash FileHash
	Shard_size types.U64
	Miner_ip   Ipv4Type
	Miner_acc  types.AccountID
}

// filler meta info
type FillerMetaInfo struct {
	Size      types.U64
	Miner_acc types.AccountID
	Hash      FileHash
}

// scheduler info
type SchedulerInfo struct {
	Ip             Ipv4Type
	StashUser      types.AccountID
	ControllerUser types.AccountID
}

type Ipv4Type_Query struct {
	Placeholder types.U8 //
	Index       types.U8
	Value       [4]types.U8
	Port        types.U16
}

type IpAddress struct {
	IPv4 Ipv4Type
	IPv6 Ipv6Type
}
type Ipv4Type struct {
	Index types.U8
	Value [4]types.U8
	Port  types.U16
}
type Ipv6Type struct {
	Index types.U8
	Value [8]types.U16
	Port  types.U16
}

// proof type
type Proof struct {
	FileId         FileHash
	Miner_pubkey   types.AccountID
	Challenge_info ChallengeInfo
	Mu             types.Bytes
	Sigma          types.Bytes
	U              types.Bytes
	HashMi         []types.Bytes
}

// challenge info
type ChallengeInfo struct {
	File_size  types.U64
	File_type  types.U8
	Block_list types.Bytes
	File_id    FileHash
	Shard_id   SliceId
	Random     []types.Bytes
}

// user space package Info
type SpacePackage struct {
	Space           types.U128
	Used_space      types.U128
	Remaining_space types.U128
	Tenancy         types.U32
	Package_type    types.U8
	Start           types.U32
	Deadline        types.U32
	State           types.Bytes
}

// proof result
type ProofResult struct {
	PublicKey types.AccountID
	FileId    FileHash
	Shard_id  SliceId
	Result    types.Bool
}

// Scheduling node public key information structure
type Chain_SchedulerPuk struct {
	Spk           [128]types.U8
	Shared_params types.Bytes
	Shared_g      [128]types.U8
}

// Proof information structure
type ProveInfo struct {
	FileId   FileHash
	MinerAcc types.AccountID
	//Cinfo    ChallengesInfo
	Mu     types.Bytes
	Sigma  types.Bytes
	U      types.Bytes
	HashMi []types.Bytes
}

type NetworkSnapshot struct {
	Total_power types.U128
	Reward      types.U128
	Random      Random
	Start       types.U32
	Deadline    types.U32
}

type Result struct {
	Sigmas []string   `json:"sigmas"`
	Tag    Tag        `json:"tag"`
	Status StatusInfo `json:"status"`
}

type Tag struct {
	T     Tag0   `json:"t"`
	MacT0 []byte `json:"mac_t0"`
}

type Tag0 struct {
	N        int64  `json:"n"`
	Enc      []byte `json:"enc"`
	FileHash []byte `json:"file_hash"`
}

type StatusInfo struct {
	StatusCode uint   `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type AutonomyFileMeta struct {
	File_hash FileHash
	File_size types.U64
	Slice     []FileHash
}

type SliceSummary struct {
	Miner_acc types.AccountID
	Signature Signature
	Message   types.Bytes
}
