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
	"github.com/CESSProject/cess-bucket/pkg/pbc"
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

type NetworkSnapshot struct {
	Total_power types.U128
	Reward      types.U128
	Random      Random
	Start       types.U32
	Deadline    types.U32
}

type Result struct {
	Sigmas []string       `json:"sigmas"`
	Tag    pbc.T          `json:"tag"`
	Status pbc.StatusInfo `json:"status"`
}

type SubmitAutonomyFileMeta struct {
	File_hash FileHash
	File_size types.U64
	Slice     []FileHash
}

type SliceSummary struct {
	Miner_acc types.AccountID `json:"miner_acc"`
	Signature Signature       `json:"signature"`
	Message   types.Bytes     `json:"message"`
}

type MessageType struct {
	ShardId   string `json:"shardId"`
	SliceHash string `json:"sliceHash"`
	MinerIp   string `json:"minerIp"` //ex:127/0/0/1/15001
}

type ChallengeReport struct {
	Message   types.Bytes
	Signature Signature
}

type AutonomyFileInfo struct {
	File_hash FileHash
	File_size types.U64
	Slice     []FileHash
	Miner_acc types.AccountID
}
